package dep

import (
	"errors"
	"io/fs"
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
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		return parts[0]
	}
	return ""
}

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
	for name, pkg := range locked.Packages {
		if !strings.HasPrefix(name, "node_modules/") {
			continue
		}
		name = strings.TrimPrefix(name, "node_modules/")
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
		parts := strings.Split(nameVersion, "@")
		if len(parts) != 2 {
			continue
		}
		name, version := parts[0], parts[1]
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
		parts := strings.Split(first, "@")
		if len(parts) != 2 {
			continue
		}
		version := parts[1]
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
