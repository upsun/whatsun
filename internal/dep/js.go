package dep

import (
	"errors"
	"io/fs"
	"strings"
)

type jsManager struct {
	fsys fs.FS
	path string

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
func newJSManager(fsys fs.FS, path string) (Manager, error) {
	m := &jsManager{
		fsys: fsys,
		path: path,
	}
	if err := m.parse(); err != nil {
		return nil, err
	}
	return m, nil
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

func (m *jsManager) Find(pattern string) ([]Dependency, error) {
	matches, err := matchDependencyKey(m.packageJSON.Dependencies, pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	var deps = make([]Dependency, 0, len(matches))
	for _, match := range matches {
		parts := strings.SplitN(match, "/", 2)
		vendor, name := "", match
		if len(parts) == 2 {
			vendor = parts[0]
		}
		deps = append(deps, Dependency{
			Vendor: vendor,
			Name:   name,
		})
	}
	return deps, nil
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
