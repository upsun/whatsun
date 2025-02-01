// Package pm detects project managers in a code directory.
package pm

import (
	"io/fs"
	"os"
	"path/filepath"

	"what/internal/match"
)

// Detect looks for evidence of package managers in a directory.
func Detect(fsys fs.FS) ([]match.Match, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}

	var filenames = make([]any, 0, len(entries))
	for _, entry := range entries {
		n := entry.Name()
		if entry.IsDir() {
			n += string(os.PathSeparator) + "."
		}
		filenames = append(filenames, n)
	}

	m := &match.Matcher{
		Rules: config.PackageManagers.Rules,
		Eval:  evalFiles,
	}

	return m.Match(filenames)
}

func evalFiles(data any, condition any) (bool, error) {
	for _, filename := range data.([]any) {
		m, err := filepath.Match(condition.(string), filename.(string))
		if err != nil {
			return false, err
		}
		if m {
			return true, nil
		}
	}
	return false, nil
}
