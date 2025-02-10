package dep

import (
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"

	"golang.org/x/mod/modfile"
)

type goManager struct {
	fsys fs.FS
	path string

	file *modfile.File
}

func newGoManager(fsys fs.FS, path string) (Manager, error) {
	b, err := fs.ReadFile(fsys, filepath.Join(path, "go.mod"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	f, err := modfile.Parse("go.mod", b, nil)
	if err != nil {
		return nil, err
	}
	return &goManager{
		fsys: fsys,
		path: path,
		file: f,
	}, nil
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

func (m *goManager) Find(pattern string) ([]Dependency, error) {
	patt, err := regexp.Compile(wildCardToRegexp(pattern))
	if err != nil {
		return nil, err
	}
	var matches []Dependency
	for _, v := range m.file.Require {
		if !v.Indirect && patt.MatchString(v.Mod.Path) {
			matches = append(matches, Dependency{
				Name:    v.Mod.Path,
				Version: v.Mod.Version,
			})
		}
	}
	return matches, nil
}
