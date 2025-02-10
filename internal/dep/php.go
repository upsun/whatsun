package dep

import (
	"errors"
	"io/fs"
	"strings"
)

type phpManager struct {
	fsys fs.FS
	path string

	composerJSON composerJSON
	composerLock composerLock
}

type composerJSON struct {
	Require map[string]string `json:"require"`
}

type composerLock struct {
	Packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"packages"`
}

func newPHPManager(fsys fs.FS, path string) (Manager, error) {
	m := &phpManager{
		fsys: fsys,
		path: path,
	}
	if err := m.parse(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *phpManager) parse() error {
	if err := parseJSON(m.fsys, m.path, "composer.json", &m.composerJSON); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := parseJSON(m.fsys, m.path, "composer.lock", &m.composerLock); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

func (m *phpManager) Find(pattern string) ([]Dependency, error) {
	matches, err := matchDependencyKey(m.composerJSON.Require, pattern)
	if err != nil {
		return nil, err
	}
	var deps = make([]Dependency, 0, len(matches))
	for _, match := range matches {
		parts := strings.SplitN(match, "/", 2)
		vendor, name := "", match
		if len(parts) == 2 {
			vendor = parts[0]
		}
		deps = append(deps, Dependency{
			Vendor:            vendor,
			Name:              name,
			VersionConstraint: m.composerJSON.Require[match],
			Version:           m.getLockedVersion(match),
		})
	}
	return deps, nil
}

func (m *phpManager) getLockedVersion(packageName string) string {
	for _, p := range m.composerLock.Packages {
		if p.Name == packageName {
			return p.Version
		}
	}
	return ""
}

func (m *phpManager) Get(name string) (Dependency, bool) {
	packageName := name
	constraint, ok := m.composerJSON.Require[name]
	if !ok && m.getLockedVersion(packageName) == "" {
		return Dependency{}, false
	}
	var vendor string
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		vendor = parts[0]
	}
	return Dependency{
		Vendor:            vendor,
		Name:              name,
		VersionConstraint: constraint,
		Version:           m.getLockedVersion(packageName),
	}, true
}
