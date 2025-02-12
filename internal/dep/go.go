package dep

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
	"golang.org/x/mod/modfile"
)

type goManager struct {
	fsys fs.FS
	path string

	initOnce sync.Once
	file     *modfile.File
}

func newGoManager(fsys fs.FS, path string) Manager {
	return &goManager{
		fsys: fsys,
		path: path,
	}
}

func (m *goManager) Init() error {
	var err error
	m.initOnce.Do(func() {
		err = m.init()
	})
	return err
}

func (m *goManager) init() error {
	b, err := fs.ReadFile(m.fsys, filepath.Join(m.path, "go.mod"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	f, err := modfile.Parse("go.mod", b, nil)
	if err != nil {
		return err
	}
	m.file = f
	return nil
}

func (m *goManager) Get(name string) (Dependency, bool) {
	for _, v := range m.file.Require {
		if v.Mod.Path == name && !v.Indirect {
			return Dependency{
				Name:    v.Mod.Path,
				Version: v.Mod.Version,
			}, true
		}
	}
	return Dependency{}, false
}

func (m *goManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for _, v := range m.file.Require {
		if !v.Indirect && wildcard.Match(pattern, v.Mod.Path) {
			deps = append(deps, Dependency{
				Name:    v.Mod.Path,
				Version: v.Mod.Version,
			})
		}
	}
	return deps
}
