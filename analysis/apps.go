package analysis

import (
	"context"
	"io/fs"
	"path/filepath"

	"what"
)

type Apps struct{}

var _ what.Analyzer = (*Apps)(nil)

func (*Apps) Name() string {
	return "apps"
}

func (*Apps) Analyze(_ context.Context, fsys fs.FS, root string) (results []what.Result, err error) {
	var seenApps = make(map[string]struct{})
	err = fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		n := d.Name()
		if isMaybeAppRootFile(n) {
			dirname := filepath.Dir(path)
			if _, ok := seenApps[dirname]; !ok {
				seenApps[dirname] = struct{}{}
				results = append(results, what.Result{Payload: dirname, Reason: n})
			}
		}
		if d.IsDir() {
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
	"composer.json": {},
	"Dockerfile":    {},
	".git":          {},
	"package.json":  {},
}

func isMaybeAppRootFile(name string) bool {
	_, ok := maybeAppRootFiles[name]
	return ok
}
