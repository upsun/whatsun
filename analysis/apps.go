package analysis

import (
	"context"
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

func (*Apps) Name() string {
	return "apps"
}

func (a *Apps) Analyze(_ context.Context, fsys fs.FS, root string) (results []what.Result, err error) {
	var seenApps = make(map[string]struct{})
	err = fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		depth := strings.Count(path, string(os.PathSeparator))
		if depth > a.MaxDepth {
			return filepath.SkipDir
		}
		if isMaybeAppRootFile(d) {
			dirname := filepath.Dir(path)
			if _, ok := seenApps[dirname]; !ok {
				seenApps[dirname] = struct{}{}
				results = append(results, what.Result{Payload: dirname, Reason: d.Name()})
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
	return
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
