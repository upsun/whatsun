package dep

import (
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
	"gopkg.in/yaml.v3"
)

type jsManager struct {
	fsys fs.FS
	path string

	initOnce sync.Once
	deps     map[string]Dependency
	toolName string
}

func newJSManager(fsys fs.FS, path string) Manager {
	return &jsManager{
		fsys: fsys,
		path: path,
	}
}

func (m *jsManager) Init() error {
	var err error
	m.initOnce.Do(func() {
		err = m.parse()
	})
	return err
}

func (m *jsManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for name, dep := range m.deps {
		if wildcard.Match(pattern, name) {
			deps = append(deps, dep)
		}
	}
	return deps
}

func (m *jsManager) Get(name string) (Dependency, bool) {
	dep, ok := m.deps[name]
	return dep, ok
}

func (m *jsManager) vendorName(name string) string {
	if strings.HasPrefix(name, "@") && strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		return strings.TrimPrefix(parts[0], "@")
	}
	return ""
}

func (m *jsManager) parse() error {
	m.deps = make(map[string]Dependency)

	// Use ReadDir to get all filenames in the directory
	entries, err := fs.ReadDir(m.fsys, m.path)
	if err != nil {
		return err
	}
	files := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		files[entry.Name()] = struct{}{}
	}

	// Handle Meteor dependencies.
	if _, ok := files[".meteor"]; ok {
		m.toolName = "meteor"
		meteorDeps, err := parseMeteorDeps(m.fsys, m.path)
		if err != nil {
			return err
		}
		for name, dep := range meteorDeps {
			dep.ToolName = m.toolName
			m.deps[name] = dep
		}
	}

	// Handle Deno dependencies.
	if _, ok := files["deno.json"]; ok {
		m.toolName = "deno"
		denoDeps, err := parseDenoDeps(m.fsys, m.path)
		if err != nil {
			return err
		}
		for name, dep := range denoDeps {
			dep.ToolName = m.toolName
			m.deps[name] = dep
		}
	}

	// For npm, pnpm, bun, and yarn, always parse package.json first for constraints.
	if _, ok := files["package.json"]; ok {
		// Detect tool based on lock files
		if _, ok := files["pnpm-lock.yaml"]; ok {
			m.toolName = "pnpm"
		} else if _, ok := files["bun.lock"]; ok {
			m.toolName = "bun"
		} else {
			m.toolName = "npm" // default for package.json
		}

		pkgJsonDeps, err := parsePackageDotJsonDeps(m.fsys, m.path, m.vendorName)
		if err != nil {
			return err
		}
		for name, dep := range pkgJsonDeps {
			dep.ToolName = m.toolName
			m.deps[name] = dep
		}

		// Then update Version fields from lock files if present.
		if _, ok := files["package-lock.json"]; ok {
			if err := parseNpmLockDeps(m.fsys, m.path, m.deps, m.vendorName, m.toolName); err != nil {
				return err
			}
		}
		if _, ok := files["pnpm-lock.yaml"]; ok {
			if err := parsePnpmLockDeps(m.fsys, m.path, m.deps, m.vendorName, m.toolName); err != nil {
				return err
			}
		}
		if _, ok := files["bun.lock"]; ok {
			if err := parseBunLockDeps(m.fsys, m.path, m.deps, m.vendorName, m.toolName); err != nil {
				return err
			}
		}
	}

	return nil
}

var npmNameVersion = regexp.MustCompile(
	`^((?:jsr:|npm:)?(?:@[\w-]+/)?[a-z0-9._](?:[a-z0-9\-.]*[a-z0-9]))?@([\dvx*]+(?:[-.](?:[\dx*]+|alpha|beta))*)`)
var denoPackageURL = regexp.MustCompile(
	`^(https://(?:deno\.land/x|esm\.sh)/[^/@]+)@([\dvx*]+(?:[-.](?:[\dx*]+|alpha|beta))*)/$`)

// parsePackageDotJsonDeps parses dependencies from package.json only (not lock files)
func parsePackageDotJsonDeps(fsys fs.FS, path string, vendorName func(string) string) (map[string]Dependency, error) {
	var npmManifest packageJSON
	err := parseJSON(fsys, path, "package.json", &npmManifest)
	if err != nil {
		return nil, err
	}
	deps := map[string]Dependency{}
	// Add regular dependencies as direct, non-dev
	for name, constraint := range npmManifest.Dependencies {
		deps[name] = Dependency{
			Constraint: constraint,
			Name:       name,
			Vendor:     vendorName(name),
			IsDirect:   true,
		}
	}
	// Add dev dependencies as direct, dev-only
	for name, constraint := range npmManifest.DevDependencies {
		deps[name] = Dependency{
			Constraint: constraint,
			Name:       name,
			Vendor:     vendorName(name),
			IsDirect:   true,
			IsDevOnly:  true,
		}
	}
	return deps, nil
}

// parseDenoDeps parses dependencies from deno.json and deno.lock
func parseDenoDeps(fsys fs.FS, path string) (map[string]Dependency, error) {
	var denoManifest denoJSON
	if err := parseJSON(fsys, path, "deno.json", &denoManifest); err != nil {
		return nil, err
	}
	deps := map[string]Dependency{}
	for _, constraint := range denoManifest.Imports {
		if strings.HasPrefix(constraint, "jsr:") || strings.HasPrefix(constraint, "npm:") {
			addDepVersion(deps, constraint, func(string) string { return "" }, true, "deno")
			continue
		}
		if matches := denoPackageURL.FindStringSubmatch(constraint); len(matches) == 3 {
			name, version := matches[1], matches[2]
			if d, ok := deps[name]; ok {
				d.Version = version
				deps[name] = d
			} else {
				deps[name] = Dependency{
					Name:     name,
					Version:  version,
					IsDirect: true,
					ToolName: "deno",
				}
			}
		}
	}
	var denoLocked denoLock
	if err := parseJSON(fsys, path, "deno.lock", &denoLocked); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for nameVersion := range denoLocked.JSR {
		addDepVersion(deps, nameVersion, func(string) string { return "" }, false, "deno")
	}
	for nameVersion := range denoLocked.NPM {
		addDepVersion(deps, nameVersion, func(string) string { return "" }, false, "deno")
	}
	return deps, nil
}

// parseNpmLockDeps updates deps with versions from package-lock.json
func parseNpmLockDeps(
	fsys fs.FS, path string, deps map[string]Dependency,
	vendorName func(string) string, toolName string,
) error {
	var locked packageLockJSON
	if err := parseJSON(fsys, path, "package-lock.json", &locked); err != nil {
		return err
	}
	for name, pkg := range locked.Dependencies {
		if d, ok := deps[name]; ok {
			// Update existing dependency (preserve IsDirect status)
			d.Version = pkg.Version
			deps[name] = d
		} else {
			// New dependency only found in lock file (indirect)
			deps[name] = Dependency{
				Name:     name,
				Version:  pkg.Version,
				Vendor:   vendorName(name),
				IsDirect: false,
				ToolName: toolName,
			}
		}
	}
	for name, pkg := range locked.Packages {
		name = strings.TrimPrefix(name, "node_modules/")
		if d, ok := deps[name]; ok {
			// Update existing dependency (preserve IsDirect status)
			d.Version = pkg.Version
			deps[name] = d
		} else {
			// New dependency only found in lock file (indirect)
			deps[name] = Dependency{
				Name:     name,
				Version:  pkg.Version,
				Vendor:   vendorName(name),
				IsDirect: false,
				ToolName: toolName,
			}
		}
	}
	return nil
}

// parsePnpmLockDeps updates deps with versions from pnpm-lock.yaml
func parsePnpmLockDeps(
	fsys fs.FS, path string, deps map[string]Dependency,
	vendorName func(string) string, toolName string,
) error {
	var pnpmLocked pnpmLockYAML
	if err := parseYAML(fsys, path, "pnpm-lock.yaml", &pnpmLocked); err != nil {
		return err
	}
	for nameVersion := range pnpmLocked.Packages {
		addDepVersion(deps, nameVersion, vendorName, false, toolName)
	}
	return nil
}

// parseBunLockDeps updates deps with versions from bun.lock
func parseBunLockDeps(
	fsys fs.FS, path string, deps map[string]Dependency,
	vendorName func(string) string, toolName string,
) error {
	var bunLocked bunLock
	if err := parseJSONC(fsys, path, "bun.lock", &bunLocked); err != nil {
		return err
	}
	for _, info := range bunLocked.Packages {
		if len(info) == 0 {
			continue
		}
		first, ok := info[0].(string)
		if !ok {
			continue
		}
		addDepVersion(deps, first, vendorName, false, toolName)
	}
	return nil
}

// addDepVersion updates the deps map with a dependency parsed from nameVersion (e.g. "foo@1.2.3").
func addDepVersion(
	deps map[string]Dependency, nameVersion string,
	vendorName func(string) string, isDirect bool, toolName string,
) {
	matches := npmNameVersion.FindStringSubmatch(nameVersion)
	if len(matches) != 3 {
		return
	}
	name, version := matches[1], matches[2]
	if d, ok := deps[name]; ok {
		// Update existing dependency (preserve IsDirect status from manifest files)
		d.Version = version
		deps[name] = d
	} else {
		// New dependency (lock file dependencies are not dev-only)
		deps[name] = Dependency{
			Name:     name,
			Version:  version,
			Vendor:   vendorName(name),
			IsDirect: isDirect,
			ToolName: toolName,
		}
	}
}

// parseMeteorDeps reads Meteor's .meteor/packages and .meteor/versions files and returns a map of dependencies.
func parseMeteorDeps(fsys fs.FS, path string) (map[string]Dependency, error) {
	var meteorPackages, meteorVersions []byte
	var meteorPackagesExists, meteorVersionsExists bool
	if b, err := fs.ReadFile(fsys, filepath.Join(path, ".meteor/packages")); err == nil {
		meteorPackages = b
		meteorPackagesExists = true
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if b, err := fs.ReadFile(fsys, filepath.Join(path, ".meteor/versions")); err == nil {
		meteorVersions = b
		meteorVersionsExists = true
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	meteorDeps := map[string]Dependency{}
	if meteorPackagesExists {
		lines := strings.Split(string(meteorPackages), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Dependencies from .meteor/packages are direct
			meteorDeps[line] = Dependency{
				Name:     line,
				IsDirect: true,
			}
		}
	}
	if meteorVersionsExists {
		lines := strings.Split(string(meteorVersions), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Format: name@version
			parts := strings.SplitN(line, "@", 2)
			if len(parts) == 2 {
				name, version := parts[0], parts[1]
				if dep, ok := meteorDeps[name]; ok {
					// Update existing dependency (preserve IsDirect status)
					dep.Version = version
					meteorDeps[name] = dep
				} else {
					// New dependency only in versions file (indirect)
					meteorDeps[name] = Dependency{
						Name:     name,
						Version:  version,
						IsDirect: false,
					}
				}
			}
		}
	}
	return meteorDeps, nil
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type denoJSON struct {
	Imports map[string]string `json:"imports"`
}

type packageLockJSON struct {
	// Version 1 (see https://docs.npmjs.com/cli/v11/configuring-npm/package-lock-json#dependencies)
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`

	// Versions 2 and 3
	Packages map[string]struct {
		Version string `json:"version"`
	} `json:"Packages"`
}

type pnpmLockYAML struct {
	Packages map[string]yaml.Node `yaml:"packages"`
}

type bunLock struct {
	Packages map[string][]any `json:"packages"`
}

type denoLock struct {
	JSR map[string]struct {
		Version string `json:"version"`
	} `json:"jsr"`
	NPM map[string]struct {
		Version string `json:"version"`
	} `json:"npm"`
}
