package dep

import (
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
	"golang.org/x/mod/modfile"
)

type goManager struct {
	fsys fs.FS
	path string

	initOnce  sync.Once
	file      *modfile.File
	bazelDeps []Dependency
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

	// Add Bazel dependencies not already direct in go.mod. Those that are
	// indirect in go.mod are replaced (see Find), as the BUILD file references
	// them directly.
	var direct []Dependency
	for _, v := range f.Require {
		if !v.Indirect {
			direct = append(direct, Dependency{Name: v.Mod.Path})
		}
	}
	withBazel, err := appendBazelDeps(direct, m.fsys, m.path, ManagerTypeGo)
	if err != nil {
		return err
	}
	m.bazelDeps = withBazel[len(direct):]
	return nil
}

func (m *goManager) Get(name string) (Dependency, bool) {
	for _, v := range m.file.Require {
		if v.Mod.Path == name && !v.Indirect {
			return Dependency{
				Name:     v.Mod.Path,
				Version:  v.Mod.Version,
				IsDirect: !v.Indirect,
				ToolName: "go",
			}, true
		}
	}
	for _, dep := range m.bazelDeps {
		if dep.Name == name {
			return dep, true
		}
	}
	return Dependency{}, false
}

func (m *goManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for _, v := range m.file.Require {
		if v.Indirect && slices.ContainsFunc(m.bazelDeps, func(d Dependency) bool { return d.Name == v.Mod.Path }) {
			continue
		}
		if wildcard.Match(pattern, v.Mod.Path) {
			deps = append(deps, Dependency{
				Name:     v.Mod.Path,
				Version:  v.Mod.Version,
				IsDirect: !v.Indirect,
				ToolName: "go",
			})
		}
	}
	for _, dep := range m.bazelDeps {
		if wildcard.Match(pattern, dep.Name) {
			deps = append(deps, dep)
		}
	}
	return deps
}
