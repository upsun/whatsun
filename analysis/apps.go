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

type Apps struct {
	SkipNested bool
	MaxDepth   int
}

var _ what.Analyzer = (*Apps)(nil)

func (*Apps) GetName() string {
	return "apps"
}

func (a *Apps) Analyze(_ context.Context, fsys fs.FS, root string) (what.Result, error) {
	var seenApps = make(map[string]struct{})
	var appList = &AppList{}
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		depth := strings.Count(path, string(os.PathSeparator))
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
			if n == "vendor" {
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
