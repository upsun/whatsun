// Package pm detects project managers in a code directory.
package pm

import (
	"fmt"
	"io/fs"
)

type DetectedPM struct {
	Category string
	Name     string
	Sources  []string
}

func (d *DetectedPM) String() string {
	return fmt.Sprintf("%s/%s (via: %s)", d.Category, d.Name, d.Sources)
}

type List []DetectedPM

// Detect looks for evidence of package managers in a directory.
func Detect(fsys fs.FS) (List, error) {
	var s store
	for fp, candidates := range config.FilePatterns {
		matches, err := fs.Glob(fsys, fp)
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			continue
		}
		for _, name := range candidates {
			if err := s.add(name, fp, len(candidates) == 1); err != nil {
				return nil, err
			}
		}
	}

	return s.list(), nil
}
