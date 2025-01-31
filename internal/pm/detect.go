// Package pm detects project managers in a code directory.
package pm

import (
	"io/fs"

	"what/internal/heuristic"
)

// Detect looks for evidence of package managers in a directory.
func Detect(fsys fs.FS) ([]heuristic.Finding, error) {
	var s heuristic.Store
	for fp, def := range config.FilePatterns {
		matches, err := fs.Glob(fsys, fp)
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			continue
		}
		s.Add(def, fp)
	}

	return s.List()
}
