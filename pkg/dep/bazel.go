package dep

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Bazel support.
//
// Bazel external dependencies are declared at the root of each workspace, in
// MODULE.bazel or WORKSPACE files. They are referenced from BUILD files in
// every package, but packages are fine-grained (often one per source
// directory), so only the declarations at the workspace root are read, in the
// same way that other manifests are.
//
// The declarations read are:
//   - Java: Maven artifacts in maven.install / maven_install / maven.artifact.
//   - Python: requirements files named in pip.parse / pip_parse / pip_install.
//   - Go: go_repository and go_deps.module (Go modules otherwise use go.mod).

const bazelToolName = "bazel"

var bazelWorkspaceFiles = []string{"MODULE.bazel", "WORKSPACE.bazel", "WORKSPACE"}

// bazelDeps holds dependencies declared at a Bazel workspace root, by manager type.
type bazelDeps map[string][]Dependency

// parseBazelDeps finds dependencies declared in Bazel workspace files in dir.
// It returns nil if dir is not a Bazel workspace root.
func parseBazelDeps(fsys fs.FS, dir string) (bazelDeps, error) {
	deps := bazelDeps{}
	found := false
	for _, name := range bazelWorkspaceFiles {
		src, err := readBazelFile(fsys, path.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if src == "" {
			continue
		}
		found = true
		if err := deps.parseWorkspaceFile(fsys, dir, stripStarlarkComments(src)); err != nil {
			return nil, err
		}
	}
	if !found {
		return nil, nil
	}
	return deps, nil
}

func (b bazelDeps) add(managerType string, d Dependency) {
	for _, existing := range b[managerType] {
		if existing.Name == d.Name {
			return
		}
	}
	d.ToolName = bazelToolName
	b[managerType] = append(b[managerType], d)
}

// readBazelFile reads a file, returning an empty string if it does not exist
// or is not a regular file (e.g. a directory named "WORKSPACE").
func readBazelFile(fsys fs.FS, name string) (string, error) {
	fi, err := fs.Stat(fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return "", nil
	}
	b, err := fs.ReadFile(fsys, name)
	return string(b), err
}

func (b bazelDeps) parseWorkspaceFile(fsys fs.FS, root, src string) error {
	for _, c := range starlarkCalls(src) {
		args := splitStarlarkArgs(c.args)
		switch c.name {
		case "maven_install", "maven.install":
			b.addMavenArtifacts(src, starlarkKwarg(args, "artifacts"))
		case "maven.artifact":
			b.addMavenArtifactCall(args)
		case "pip.parse", "pip_parse", "pip_install":
			if err := b.addPythonRequirements(fsys, root, args); err != nil {
				return err
			}
		case "go_repository":
			b.addGoModule(args, "importpath")
		case "go_deps.module":
			b.addGoModule(args, "path")
		}
	}
	return nil
}

// addMavenArtifacts adds artifacts from a maven_install "artifacts" expression.
// The expression may be a list, the name of a variable defined in src, or a
// concatenation of those (e.g. "BASE + [...]").
func (b bazelDeps) addMavenArtifacts(src, expr string) {
	for _, term := range starlarkTerms(src, expr) {
		if !strings.HasPrefix(term, "[") {
			continue
		}
		for _, elem := range splitStarlarkArgs(term[1:matchStarlarkBracket(term, 0)]) {
			if calls := starlarkCalls(elem); len(calls) == 1 && calls[0].name == "maven.artifact" {
				b.addMavenArtifactCall(splitStarlarkArgs(calls[0].args))
				continue
			}
			// Only plain string elements are coordinates: strings nested
			// elsewhere (e.g. exclusions) are not.
			if elem == "" || elem[0] != '"' && elem[0] != '\'' {
				continue
			}
			coord := firstString(elem)
			if strings.ContainsAny(coord, "/ ") {
				continue
			}
			// Coordinates are either group:artifact[:packaging[:classifier]]:version,
			// or group:artifact:version[:classifier]@packaging.
			coord, _, gradleStyle := strings.Cut(coord, "@")
			parts := strings.Split(coord, ":")
			if len(parts) < 2 {
				continue
			}
			var version string
			switch {
			case gradleStyle && len(parts) >= 3:
				version = parts[2]
			case !gradleStyle && len(parts) >= 3:
				version = parts[len(parts)-1]
			}
			b.addMavenArtifact(parts[0], parts[1], version)
		}
	}
}

// addMavenArtifactCall adds an artifact from maven.artifact() arguments,
// which may be keyword or positional (group, artifact, version).
func (b bazelDeps) addMavenArtifactCall(args []string) {
	group, artifact, version := firstString(starlarkKwarg(args, "group")),
		firstString(starlarkKwarg(args, "artifact")), firstString(starlarkKwarg(args, "version"))
	if group == "" && len(args) >= 3 {
		group, artifact, version = firstString(args[0]), firstString(args[1]), firstString(args[2])
	}
	b.addMavenArtifact(group, artifact, version)
}

func (b bazelDeps) addMavenArtifact(group, artifact, version string) {
	if group == "" || artifact == "" {
		return
	}
	b.add(ManagerTypeJava, Dependency{
		Vendor:     group,
		Name:       group + ":" + artifact,
		Constraint: version,
		Version:    version,
		IsDirect:   true,
	})
}

var bazelRequirementsAttrs = []string{
	"requirements_lock",
	"requirements_by_platform",
	"requirements_linux",
	"requirements_darwin",
	"requirements_windows",
	"requirements", // pip_install
}

// addPythonRequirements adds dependencies from the requirements files named in
// a pip.parse (or similar) call.
func (b bazelDeps) addPythonRequirements(fsys fs.FS, root string, args []string) error {
	for _, attr := range bazelRequirementsAttrs {
		for _, label := range bazelRequirementsLabels(starlarkKwarg(args, attr)) {
			file, ok := bazelLabelPath(label)
			if !ok {
				continue
			}
			src, err := readBazelFile(fsys, path.Join(root, file))
			if err != nil {
				return err
			}
			reqs, err := parseRequirementsTXT(bytes.NewReader([]byte(src)), bazelToolName)
			if err != nil {
				return err
			}
			for _, d := range reqs {
				b.add(ManagerTypePython, d)
			}
		}
	}
	return nil
}

// bazelRequirementsLabels returns the labels in a requirements attribute: a
// string, or a dict whose keys are labels (e.g. requirements_by_platform).
func bazelRequirementsLabels(expr string) []string {
	if !strings.HasPrefix(expr, "{") {
		return starlarkStrings(expr)
	}
	var labels []string
	for _, entry := range splitStarlarkArgs(expr[1:matchStarlarkBracket(expr, 0)]) {
		if key := firstString(entry); key != "" {
			labels = append(labels, key)
		}
	}
	return labels
}

// bazelLabelPath converts a label for a file in the main repository (e.g.
// "//tools:requirements.txt") to a path relative to the workspace root.
func bazelLabelPath(label string) (string, bool) {
	if strings.HasPrefix(label, "@") {
		return "", false
	}
	pkg, target, ok := strings.Cut(strings.TrimPrefix(label, "//"), ":")
	if !ok {
		return label, label != ""
	}
	return path.Join(pkg, target), target != ""
}

func (b bazelDeps) addGoModule(args []string, pathKey string) {
	d := Dependency{Name: firstString(starlarkKwarg(args, pathKey)), IsDirect: true}
	if d.Name == "" {
		return
	}
	for _, key := range []string{"version", "tag", "commit"} {
		if v := firstString(starlarkKwarg(args, key)); v != "" {
			d.Version = v
			break
		}
	}
	b.add(ManagerTypeGo, d)
}

var pythonNameSeparators = regexp.MustCompile(`[-_.]+`)

// normalizePythonName normalizes a Python package name (see PEP 503).
func normalizePythonName(name string) string {
	return pythonNameSeparators.ReplaceAllString(strings.ToLower(name), "-")
}

// appendBazelDeps appends the Bazel dependencies for managerType to deps,
// skipping any that already exist (from other manifests) by name.
func appendBazelDeps(deps []Dependency, fsys fs.FS, dir, managerType string) ([]Dependency, error) {
	bd, err := parseBazelDeps(fsys, dir)
	if err != nil {
		return nil, err
	}
	normalize := strings.ToLower
	if managerType == ManagerTypePython {
		normalize = normalizePythonName
	}
	for _, d := range bd[managerType] {
		name := normalize(d.Name)
		if !slices.ContainsFunc(deps, func(e Dependency) bool { return normalize(e.Name) == name }) {
			deps = append(deps, d)
		}
	}
	return deps, nil
}
