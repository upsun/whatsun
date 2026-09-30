package dep

import (
	"errors"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
)

// Bazel support.
//
// Bazel does not have a single manifest of dependencies: external repositories
// are declared at the workspace root (in MODULE.bazel or WORKSPACE), and
// referenced by label from rules in BUILD files. The labels are escaped forms
// of the real package names, e.g. "@maven//:com_google_guava_guava".
//
// Dependencies are therefore found by reading the declarations at the
// workspace root, and resolving labels in the directory's BUILD file against
// them. Labels that cannot be resolved reliably are skipped, rather than
// guessed.

const bazelToolName = "bazel"

var (
	bazelBuildFiles     = []string{"BUILD.bazel", "BUILD"}
	bazelWorkspaceFiles = []string{"MODULE.bazel", "WORKSPACE.bazel", "WORKSPACE"}
)

// bazelDeps holds dependencies found in Bazel files, by manager type.
type bazelDeps map[string][]Dependency

// bazelWorkspace holds dependency declarations from the workspace root.
type bazelWorkspace struct {
	isRoot bool // Whether the workspace root is the directory being analyzed.

	mavenRepos     map[string]struct{}   // Names of Maven repositories (default "maven").
	mavenArtifacts map[string]Dependency // Maven artifacts keyed by escaped label.
	mavenOrder     []string              // Escaped labels in declaration order.

	pythonHubs map[string]struct{} // Names of pip hub repositories.

	goRepos map[string]Dependency // Go modules keyed by repository name.
}

// parseBazelDeps finds dependencies declared in Bazel files for the directory.
// It returns nil if the directory is not part of a Bazel workspace.
func parseBazelDeps(fsys fs.FS, dir string) (bazelDeps, error) {
	buildSrc, err := readFirst(fsys, dir, bazelBuildFiles)
	if err != nil {
		return nil, err
	}
	// Only look in parent directories if there is a BUILD file here, to avoid
	// repeated lookups for directories unrelated to Bazel.
	root, err := findBazelRoot(fsys, dir, buildSrc != "")
	if err != nil {
		return nil, err
	}
	if root == "" && buildSrc == "" {
		return nil, nil
	}
	ws, err := parseBazelWorkspace(fsys, root)
	if err != nil {
		return nil, err
	}
	ws.isRoot = root == path.Clean(dir)

	deps := bazelDeps{}
	if ws.isRoot {
		for _, label := range ws.mavenOrder {
			deps.add(ManagerTypeJava, ws.mavenArtifacts[label])
		}
	}
	for _, c := range starlarkCalls(stripStarlarkComments(buildSrc)) {
		lang := bazelRuleLanguage(c.name)
		if lang == "" {
			continue
		}
		args := splitStarlarkArgs(c.args)
		for _, key := range bazelDepAttrs(lang) {
			expr := starlarkKwarg(args, key)
			if expr == "" {
				continue
			}
			if lang == ManagerTypePython {
				for _, call := range starlarkCalls(expr) {
					if call.name == "requirement" {
						if lits := starlarkStrings(call.args); len(lits) > 0 {
							deps.add(lang, Dependency{Name: normalizePythonName(lits[0]), IsDirect: true})
						}
					}
				}
			}
			for _, label := range starlarkStrings(expr) {
				if d, ok := ws.resolveLabel(lang, label); ok {
					deps.add(lang, d)
				}
			}
		}
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

// readFirst reads the first file that exists out of the given names in dir.
// It returns an empty string if none exist.
func readFirst(fsys fs.FS, dir string, names []string) (string, error) {
	for _, name := range names {
		b, err := fs.ReadFile(fsys, path.Join(dir, name))
		if err == nil {
			return string(b), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
	}
	return "", nil
}

// findBazelRoot looks for the workspace root in dir and, optionally, its parents.
// It returns an empty string if none is found.
func findBazelRoot(fsys fs.FS, dir string, parents bool) (string, error) {
	dir = path.Clean(dir)
	for {
		for _, name := range bazelWorkspaceFiles {
			_, err := fs.Stat(fsys, path.Join(dir, name))
			if err == nil {
				return dir, nil
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return "", err
			}
		}
		if !parents || dir == "." || dir == "/" {
			return "", nil
		}
		dir = path.Dir(dir)
	}
}

func parseBazelWorkspace(fsys fs.FS, root string) (*bazelWorkspace, error) {
	ws := &bazelWorkspace{
		mavenRepos:     map[string]struct{}{"maven": {}},
		mavenArtifacts: map[string]Dependency{},
		pythonHubs:     map[string]struct{}{"pip": {}, "pypi": {}},
		goRepos:        map[string]Dependency{},
	}
	if root == "" {
		return ws, nil
	}

	for _, name := range bazelWorkspaceFiles {
		b, err := fs.ReadFile(fsys, path.Join(root, name))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		ws.parseFile(stripStarlarkComments(string(b)))
	}

	// Go dependencies are usually managed via go.mod (e.g. with Gazelle's
	// go_deps.from_file), and referenced by their Gazelle repository names.
	b, err := fs.ReadFile(fsys, path.Join(root, "go.mod"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if len(b) > 0 {
		if f, err := modfile.Parse("go.mod", b, nil); err == nil {
			for _, r := range f.Require {
				ws.addGoRepo(Dependency{Name: r.Mod.Path, Version: r.Mod.Version, IsDirect: !r.Indirect})
			}
		}
	}

	return ws, nil
}

func (ws *bazelWorkspace) parseFile(src string) {
	for _, c := range starlarkCalls(src) {
		args := splitStarlarkArgs(c.args)
		switch c.name {
		case "maven_install", "maven.install":
			if name := firstString(starlarkKwarg(args, "name")); name != "" {
				ws.mavenRepos[name] = struct{}{}
			}
			ws.addMavenArtifacts(src, starlarkKwarg(args, "artifacts"))
		case "maven.artifact":
			group, artifact, version := firstString(starlarkKwarg(args, "group")),
				firstString(starlarkKwarg(args, "artifact")), firstString(starlarkKwarg(args, "version"))
			ws.addMavenArtifact(group, artifact, version)
		case "pip.parse", "pip_parse", "pip_install":
			for _, key := range []string{"hub_name", "name"} {
				if name := firstString(starlarkKwarg(args, key)); name != "" {
					ws.pythonHubs[name] = struct{}{}
				}
			}
		case "go_repository":
			ws.addGoRepoFromArgs(args, "name", "importpath")
		case "go_deps.module":
			ws.addGoRepoFromArgs(args, "", "path")
		}
	}
}

// addMavenArtifacts adds artifacts from a maven_install "artifacts" expression.
// The expression may be a list, or the name of a variable defined in src.
func (ws *bazelWorkspace) addMavenArtifacts(src, expr string) {
	if isStarlarkIdent(expr) {
		expr = starlarkAssignment(src, expr)
	}
	for _, c := range starlarkCalls(expr) {
		if c.name == "maven.artifact" {
			args := splitStarlarkArgs(c.args)
			group, artifact, version := firstString(starlarkKwarg(args, "group")),
				firstString(starlarkKwarg(args, "artifact")), firstString(starlarkKwarg(args, "version"))
			if group == "" && len(args) >= 3 {
				group, artifact, version = firstString(args[0]), firstString(args[1]), firstString(args[2])
			}
			ws.addMavenArtifact(group, artifact, version)
		}
	}
	for _, coord := range starlarkStrings(expr) {
		// Coordinates: group:artifact[:packaging[:classifier]]:version
		parts := strings.Split(coord, ":")
		if len(parts) < 2 || strings.ContainsAny(coord, "/@ ") {
			continue
		}
		var version string
		if len(parts) >= 3 {
			version = parts[len(parts)-1]
		}
		ws.addMavenArtifact(parts[0], parts[1], version)
	}
}

func (ws *bazelWorkspace) addMavenArtifact(group, artifact, version string) {
	if group == "" || artifact == "" {
		return
	}
	label := escapeBazelName(group + ":" + artifact)
	if _, ok := ws.mavenArtifacts[label]; !ok {
		ws.mavenOrder = append(ws.mavenOrder, label)
	}
	ws.mavenArtifacts[label] = Dependency{
		Vendor:     group,
		Name:       group + ":" + artifact,
		Constraint: version,
		Version:    version,
		IsDirect:   true,
	}
}

func (ws *bazelWorkspace) addGoRepoFromArgs(args []string, nameKey, pathKey string) {
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
	ws.addGoRepo(d)
	if nameKey != "" {
		if name := firstString(starlarkKwarg(args, nameKey)); name != "" {
			ws.goRepos[name] = d
		}
	}
}

func (ws *bazelWorkspace) addGoRepo(d Dependency) {
	ws.goRepos[goRepoName(d.Name)] = d
}

// resolveLabel converts a label referenced in a rule into a dependency.
func (ws *bazelWorkspace) resolveLabel(managerType, label string) (Dependency, bool) {
	repo, pkg, target := splitBazelLabel(label)
	switch managerType {
	case ManagerTypeJava:
		if _, ok := ws.mavenRepos[repo]; !ok {
			return Dependency{}, false
		}
		escaped := strings.ToLower(target)
		if d, ok := ws.mavenArtifacts[escaped]; ok {
			return d, true
		}
		return guessMavenArtifact(escaped)

	case ManagerTypePython:
		if _, ok := ws.pythonHubs[repo]; !ok || pkg == "" {
			return Dependency{}, false
		}
		name, _, _ := strings.Cut(pkg, "/")
		return Dependency{Name: normalizePythonName(name), IsDirect: true}, true

	case ManagerTypeJavaScript:
		// rules_js: "//:node_modules/lodash"
		if name, ok := strings.CutPrefix(target, "node_modules/"); ok && name != "" && !strings.HasPrefix(name, ".") {
			return Dependency{Name: name, Vendor: jsVendor(name), IsDirect: true}, true
		}
		// rules_nodejs: "@npm//lodash" or "@npm//@types/node"
		if repo == "npm" && pkg != "" {
			return Dependency{Name: pkg, Vendor: jsVendor(pkg), IsDirect: true}, true
		}

	case ManagerTypeGo:
		if d, ok := ws.goRepos[repo]; ok {
			// Referenced directly from the BUILD file, even if indirect in go.mod.
			d.IsDirect = true
			return d, true
		}
	}
	return Dependency{}, false
}

// splitBazelLabel splits a label like "@repo//pkg/path:target" into its parts.
// The target defaults to the last component of the package path.
func splitBazelLabel(label string) (repo, pkg, target string) {
	rest := label
	if strings.HasPrefix(rest, "@") {
		rest = strings.TrimLeft(rest, "@")
		var ok bool
		repo, rest, ok = strings.Cut(rest, "//")
		if !ok {
			return rest, "", rest
		}
	} else {
		rest = strings.TrimPrefix(rest, "//")
	}
	pkg, target, ok := strings.Cut(rest, ":")
	if !ok {
		target = path.Base(pkg)
	}
	return repo, pkg, target
}

// knownMavenGroups are used to interpret Maven labels that cannot be resolved
// against the workspace's artifact declarations, e.g. if they are declared in
// a lock file or a macro. Longer (more specific) groups must come first.
var knownMavenGroups = []string{
	"com.fasterxml.jackson.core",
	"com.fasterxml.jackson.datatype",
	"com.google.code.gson",
	"com.google.guava",
	"com.google.protobuf",
	"com.typesafe.play",
	"io.micronaut",
	"io.quarkus",
	"io.jooby",
	"io.grpc",
	"io.netty",
	"org.apache.commons",
	"org.grails",
	"org.junit.jupiter",
	"org.slf4j",
	"org.springframework.boot",
	"org.springframework",
}

func guessMavenArtifact(escaped string) (Dependency, bool) {
	for _, group := range knownMavenGroups {
		if artifact, ok := strings.CutPrefix(escaped, escapeBazelName(group)+"_"); ok && artifact != "" {
			return Dependency{
				Vendor:   group,
				Name:     group + ":" + strings.ReplaceAll(artifact, "_", "-"),
				IsDirect: true,
			}, true
		}
	}
	return Dependency{}, false
}

// bazelRuleLanguage returns the manager type for a Bazel rule name.
func bazelRuleLanguage(rule string) string {
	switch {
	case strings.HasPrefix(rule, "java_"), strings.HasPrefix(rule, "kt_jvm_"),
		strings.HasPrefix(rule, "scala_"), rule == "springboot":
		return ManagerTypeJava
	case strings.HasPrefix(rule, "py_"):
		return ManagerTypePython
	case strings.HasPrefix(rule, "go_"):
		return ManagerTypeGo
	case strings.HasPrefix(rule, "js_"), strings.HasPrefix(rule, "ts_"),
		strings.HasPrefix(rule, "nodejs_"), strings.HasPrefix(rule, "npm_"):
		return ManagerTypeJavaScript
	}
	return ""
}

// bazelDepAttrs returns the rule attributes that may reference dependencies.
func bazelDepAttrs(managerType string) []string {
	if managerType == ManagerTypeJavaScript {
		return []string{"deps", "data"}
	}
	if managerType == ManagerTypeJava {
		return []string{"deps", "runtime_deps", "exports"}
	}
	return []string{"deps"}
}

// escapeBazelName converts a name to the form used in repository and target
// names, by replacing non-alphanumeric characters with underscores.
func escapeBazelName(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return '_'
	}, s)
}

// goRepoName returns the Gazelle repository name for a Go module path, e.g.
// "github.com/gorilla/mux" becomes "com_github_gorilla_mux".
func goRepoName(modulePath string) string {
	host, rest, _ := strings.Cut(modulePath, "/")
	labels := strings.Split(host, ".")
	for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
		labels[i], labels[j] = labels[j], labels[i]
	}
	name := strings.Join(labels, ".")
	if rest != "" {
		name += "/" + rest
	}
	return escapeBazelName(name)
}

var pythonNameSeparators = regexp.MustCompile(`[-_.]+`)

// normalizePythonName normalizes a Python package name (see PEP 503).
func normalizePythonName(name string) string {
	return pythonNameSeparators.ReplaceAllString(strings.ToLower(name), "-")
}

func jsVendor(name string) string {
	if scope, _, ok := strings.Cut(name, "/"); ok && strings.HasPrefix(scope, "@") {
		return strings.TrimPrefix(scope, "@")
	}
	return ""
}

// appendBazelDeps appends the Bazel dependencies for managerType to deps,
// skipping any that already exist (from other manifests) by name.
func appendBazelDeps(deps []Dependency, fsys fs.FS, dir, managerType string) ([]Dependency, error) {
	bd, err := parseBazelDeps(fsys, dir)
	if err != nil {
		return nil, err
	}
	for _, d := range bd[managerType] {
		if !slices.ContainsFunc(deps, func(e Dependency) bool { return strings.EqualFold(e.Name, d.Name) }) {
			deps = append(deps, d)
		}
	}
	return deps, nil
}
