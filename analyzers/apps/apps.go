package apps

import (
	"context"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/cel-go/cel"

	"what"
	"what/internal/eval"
	"what/internal/eval/celfuncs"
	"what/internal/match"
	"what/internal/pm"
)

//go:embed expr.cache
var exprCache []byte

type Analyzer struct {
	AllowNested bool
	MaxDepth    int
}

func (*Analyzer) String() string {
	return "apps"
}

func (a *Analyzer) Analyze(_ context.Context, fsys fs.FS) (what.Result, error) {
	cache, err := eval.NewFileCacheWithContent(exprCache, "")
	if err != nil {
		return nil, err
	}
	dot := "."
	evRoot := &dot
	var celOptions []cel.EnvOption
	celOptions = append(celOptions, celfuncs.AllFileFunctions(&fsys, evRoot)...)
	celOptions = append(celOptions, celfuncs.AllComposerFunctions(&fsys, evRoot)...)
	celOptions = append(
		celOptions,
		celfuncs.JSONQueryStringCELFunction(),
		celfuncs.VersionParse(),
	)

	ev, err := eval.NewEvaluator(&eval.Config{
		Cache:      cache,
		EnvOptions: append(celOptions, celfuncs.AllComposerFunctions(&fsys, evRoot)...),
	})
	if err != nil {
		return nil, err
	}

	var seenApps = make(map[string]App)
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate depth.
		// In a structure ./a/b/c then files in the root are level 0, in "a" are in level 1, in "b" are level 2, etc.
		var depth int
		if path != "." {
			depth = strings.Count(path, string(os.PathSeparator))
			if d.IsDir() {
				depth++
			}
		}

		var (
			foundAppPath         string
			foundPackageManagers []match.Match
		)

		if d.Name() == ".platform.app.yaml" {
			foundAppPath = filepath.Dir(path)
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
		}

		// Look for directories that have identifiable package managers.
		// Only do this at the top two levels to avoid false positives.
		if depth <= 1 && d.IsDir() {
			*evRoot = path
			pms, err := pm.Detect(ev)
			if err != nil {
				return err
			}
			if len(pms) > 0 {
				foundAppPath = path
				foundPackageManagers = pms
			}
		}

		// Add found apps to the list.
		if foundAppPath != "" {
			if seen, ok := seenApps[foundAppPath]; ok {
				if foundPackageManagers != nil {
					seen.PackageManagers = foundPackageManagers
					seenApps[foundAppPath] = seen
				}
			} else {
				seenApps[foundAppPath] = App{Dir: foundAppPath, PackageManagers: foundPackageManagers}
			}
			if !a.AllowNested && depth > 0 {
				return filepath.SkipDir
			}
		}

		if depth > a.MaxDepth {
			return filepath.SkipDir
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
	PackageManagers []match.Match
}

type List []App

func (l List) String() string {
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
