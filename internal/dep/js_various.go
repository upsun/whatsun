package dep

import (
	"errors"
	"io/fs"
	"strings"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
)

type jsManager struct {
	fsys fs.FS
	path string

	initOnce        sync.Once
	packageJSON     packageJSON
	packageLockJSON packageLockJSON
}

type packageJSON struct {
	Dependencies map[string]string `json:"dependencies"`
}

type packageLockJSON struct {
	Packages map[string]struct {
		Version string `json:"version"`
	} `json:"Packages"`
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

func (m *jsManager) parse() error {
	if err := parseJSON(m.fsys, m.path, "package.json", &m.packageJSON); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := parseJSON(m.fsys, m.path, "package-lock.json", &m.packageLockJSON); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

func (m *jsManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for name := range m.packageJSON.Dependencies {
		if wildcard.Match(pattern, name) {
			parts := strings.SplitN(name, "/", 2)
			var vendor string
			if len(parts) == 2 {
				vendor = parts[0]
			}
			deps = append(deps, Dependency{
				Vendor: vendor,
				Name:   name,
			})
		}
	}
	return deps
}

func (m *jsManager) getLockedVersion(packageName string) string {
	if pkg, ok := m.packageLockJSON.Packages["node_modules/"+packageName]; ok {
		return pkg.Version
	}
	return ""
}

func (m *jsManager) Get(name string) (Dependency, bool) {
	constraint, ok := m.packageJSON.Dependencies[name]
	if !ok {
		return Dependency{}, false
	}
	var vendor string
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		vendor = parts[0]
	}
	return Dependency{
		Name:       name,
		Vendor:     vendor,
		Constraint: constraint,
		Version:    m.getLockedVersion(name),
	}, true
}
