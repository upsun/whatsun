package dep

import (
	"errors"
	"io/fs"
	"strings"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
)

type phpManager struct {
	fsys fs.FS
	path string

	composerJSON composerJSON
	composerLock composerLock

	initOnce sync.Once
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

func newPHPManager(fsys fs.FS, path string) Manager {
	return &phpManager{
		fsys: fsys,
		path: path,
	}
}

func (m *phpManager) Init() error {
	var err error
	m.initOnce.Do(func() {
		err = parseJSON(m.fsys, m.path, "composer.json", &m.composerJSON)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				err = nil
			}
			return
		}
		err = parseJSON(m.fsys, m.path, "composer.lock", &m.composerLock)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				err = nil
			}
			return
		}
	})
	return err
}

func (m *phpManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for name, constraint := range m.composerJSON.Require {
		if wildcard.Match(pattern, name) {
			parts := strings.SplitN(name, "/", 2)
			var vendor string
			if len(parts) == 2 {
				vendor = parts[0]
			}
			deps = append(deps, Dependency{
				Vendor:     vendor,
				Name:       name,
				Constraint: constraint,
				Version:    m.getLockedVersion(name),
			})
		}
	}
	return deps
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
		Vendor:     vendor,
		Name:       name,
		Constraint: constraint,
		Version:    m.getLockedVersion(packageName),
	}, true
}
