package analysis

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"what"
)

type AppAnalyzer struct {
	SkipNested bool
	MaxDepth   int
}

var _ what.Analyzer = (*AppAnalyzer)(nil)

func (*AppAnalyzer) GetName() string {
	return "apps"
}

func (a *AppAnalyzer) Analyze(_ context.Context, fsys fs.FS, root string) (what.Result, error) {
	var seenApps = make(map[string]struct{})
	var appList = &AppList{}
	var initialDepth = strings.Count(root, string(os.PathSeparator))
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		depth := strings.Count(path, string(os.PathSeparator)) - initialDepth
		if depth > a.MaxDepth {
			return filepath.SkipDir
		}
		if isMaybeAppRootFile(d) {
			dirname := filepath.Dir(path)
			if _, ok := seenApps[dirname]; !ok {
				seenApps[dirname] = struct{}{}
				// TODO where do we want to expose the reason behind each path?
				appList.Paths = append(appList.Paths, dirname)
			}
			if a.SkipNested {
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
			case "vendor", "node_modules", "packages", "pkg", "tests", "logs", "doc", "docs", "bin", "dist":
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return appList, err
}

var maybeAppRootFiles = map[string]struct{}{
	"composer.lock":     {},
	"package-lock.json": {},
	"Cargo.lock":        {},
	"yarn.lock":         {},
	"pubspec.lock":      {},
	"Podfile.lock":      {},
	"pnpm-lock.yaml":    {},

	".platform.app.yaml": {},
}

func isMaybeAppRootFile(f fs.DirEntry) bool {
	_, ok := maybeAppRootFiles[f.Name()]
	if !ok {
		return false
	}
	if !f.IsDir() {
		fi, err := f.Info()
		if err == nil && fi.Size() == 0 {
			return false
		}
	}
	return true
}

type AppList struct {
	Paths []string
}

func (a *AppList) GetSummary() string {
	var s string
	switch len(a.Paths) {
	case 0:
		return "No probable app directories detected"
	case 1:
		s += "1 probable app directory detected:"
	default:
		s += fmt.Sprintf("%d probable app directories detected:", len(a.Paths))
	}
	for _, path := range a.Paths {
		s += "\n  " + path
	}
	return s
}
