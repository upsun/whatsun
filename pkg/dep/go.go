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

	// Parse Bazel dependencies if Bazel files are present
	if HasBazelFiles(m.fsys, m.path) {
		bazelParser, err := ParseBazelDependencies(m.fsys, m.path)
		if err != nil {
			return err
		}
		m.bazelDeps = bazelParser.GetGoDeps()
	}

	return nil
}

func (m *goManager) Get(name string) (Dependency, bool) {
	if m.file != nil {
		// Check go.mod dependencies first
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
	}

	// Check Bazel dependencies
	for _, dep := range m.bazelDeps {
		if dep.Name == name {
			return dep, true
		}
	}

	return Dependency{}, false
}

func (m *goManager) Find(pattern string) []Dependency {
	var deps []Dependency
	seen := make(map[string]struct{})

	// Add go.mod dependencies
	if m.file != nil {
		for _, v := range m.file.Require {
			if !v.Indirect && wildcard.Match(pattern, v.Mod.Path) {
				deps = append(deps, Dependency{
					Name:     v.Mod.Path,
					Version:  v.Mod.Version,
					IsDirect: !v.Indirect,
					ToolName: "go",
				})
				seen[v.Mod.Path] = struct{}{}
			}
		}
	}

	// Add Bazel dependencies (avoid duplicates)
	for _, dep := range m.bazelDeps {
		if _, exists := seen[dep.Name]; !exists && wildcard.Match(pattern, dep.Name) {
			deps = append(deps, dep)
		}
	}

	return deps
}
