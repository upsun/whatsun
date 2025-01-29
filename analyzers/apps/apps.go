package apps

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"what"
	"what/internal/pm"
)

type Analyzer struct {
	AllowNested bool
	MaxDepth    int
}

func New() what.Analyzer {
	return &Analyzer{MaxDepth: 3}
}

func (*Analyzer) GetName() string {
	return "apps"
}

func (a *Analyzer) Analyze(_ context.Context, fsys fs.FS) (what.Result, error) {
	var seenApps = make(map[string]App)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		depth := strings.Count(path, string(os.PathSeparator))
		if depth > a.MaxDepth {
			return filepath.SkipDir
		}
		if d.Name() == ".platform.app.yaml" {
			dirname := filepath.Dir(path)
			if _, ok := seenApps[dirname]; !ok {
				seenApps[dirname] = App{Dir: dirname}
			}
			// Skip nested apps below the top level.
			if !a.AllowNested && depth > 0 {
				return filepath.SkipDir
			}
		}
		if d.IsDir() {
			n := d.Name()
			// TODO implement actual gitignore
			if len(n) > 1 && n[0] == '.' {
				return filepath.SkipDir
			}
			switch n {
			case "vendor", "node_modules", "packages", "pkg", "tests", "logs", "doc", "docs", "bin", "dist",
				"__pycache__", "venv", "virtualenv", "target", "out", "build", "obj", "elm-stuff":
				return filepath.SkipDir
			}

			subFS, err := fs.Sub(fsys, path)
			if err != nil {
				return err
			}

			pms, err := pm.Detect(subFS)
			if err != nil {
				return err
			}
			if len(pms) > 0 {
				if seen, ok := seenApps[path]; ok {
					seen.PackageManagers = pms
				} else {
					seenApps[path] = App{Dir: path, PackageManagers: pms}
				}
				if !a.AllowNested && path != "." {
					return filepath.SkipDir
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var list = make(List, len(seenApps))
	i := 0
	for _, app := range seenApps {
		list[i] = app
		i++
	}

	slices.SortFunc(list, func(a, b App) int {
		return strings.Compare(a.Dir, b.Dir)
	})

	return list, err
}

type App struct {
	Dir             string
	PackageManagers pm.List
}

type List []App

func (l List) GetSummary() string {
	var s string
	switch len(l) {
	case 0:
		return "No probable app directories detected"
	case 1:
		s += "1 probable app directory detected:"
	default:
		s += fmt.Sprintf("%d probable app directories detected:", len(l))
	}
	for _, a := range l {
		s += fmt.Sprintf("\n  %+v", a)
	}

	return s
}
