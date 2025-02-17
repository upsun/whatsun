package dep

import (
	"errors"
	"io/fs"
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
}

type packageJSON struct {
	Dependencies map[string]string `json:"dependencies"`
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

// TODO can any repetition be avoided between jsManager and phpManager?
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

func (m *jsManager) vendorName(name string) string {
	if strings.HasPrefix(name, "@") && strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		return strings.TrimPrefix(parts[0], "@")
	}
	return ""
}

var npmNameVersion = regexp.MustCompile(`^((?:@[\w-]+/)?[a-z0-9._](?:[a-z0-9\-.]*[a-z0-9]))?@([\dvx*]+(?:[-.](?:[\dx*]+|alpha|beta))*)`)

func (m *jsManager) parse() error {
	var manifest packageJSON
	if err := parseJSON(m.fsys, m.path, "package.json", &manifest); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	m.deps = map[string]Dependency{}
	for name, constraint := range manifest.Dependencies {
		m.deps[name] = Dependency{Constraint: constraint, Name: name, Vendor: m.vendorName(name)}
	}

	var locked packageLockJSON
	if err := parseJSON(m.fsys, m.path, "package-lock.json", &locked); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for name, pkg := range locked.Dependencies {
		if d, ok := m.deps[name]; ok {
			d.Version = pkg.Version
			m.deps[name] = d
		} else {
			m.deps[name] = Dependency{Name: name, Version: pkg.Version, Vendor: m.vendorName(name)}
		}
	}
	for name, pkg := range locked.Packages {
		if strings.HasPrefix(name, "node_modules/") {
			name = strings.TrimPrefix(name, "node_modules/")
		}
		if d, ok := m.deps[name]; ok {
			d.Version = pkg.Version
			m.deps[name] = d
		} else {
			m.deps[name] = Dependency{Name: name, Version: pkg.Version, Vendor: m.vendorName(name)}
		}
	}

	var pnpmLocked pnpmLockYAML
	if err := parseYAML(m.fsys, m.path, "pnpm-lock.yaml", &pnpmLocked); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for nameVersion := range pnpmLocked.Packages {
		matches := npmNameVersion.FindStringSubmatch(nameVersion)
		if len(matches) != 3 {
			continue
		}
		name, version := matches[1], matches[2]
		if d, ok := m.deps[name]; ok {
			d.Version = version
			m.deps[name] = d
		} else {
			m.deps[name] = Dependency{Name: name, Version: version, Vendor: m.vendorName(name)}
		}
	}

	var bunLocked bunLock
	if err := parseJSONC(m.fsys, m.path, "bun.lock", &bunLocked); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for name, info := range bunLocked.Packages {
		if len(info) == 0 {
			continue
		}
		first, ok := info[0].(string)
		if !ok {
			continue
		}
		matches := npmNameVersion.FindStringSubmatch(first)
		if len(matches) != 3 {
			continue
		}
		version := matches[2]
		if d, ok := m.deps[name]; ok {
			d.Version = version
			m.deps[name] = d
		} else {
			m.deps[name] = Dependency{Name: name, Version: version, Vendor: m.vendorName(name)}
		}
	}

	return nil
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
