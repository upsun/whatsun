// Package pm detects project managers in a code directory.
package pm

import (
	"io/fs"

	"what/internal/match"
)

// Detect looks for evidence of package managers in a directory.
func Detect(fsys fs.FS) ([]match.Match, error) {
	r, err := rules()
	if err != nil {
		return nil, err
	}

	return (&match.Matcher{
		Rules: r,
		Eval:  evalFiles,
	}).Match(fsys)
}

func evalFiles(data any, condition string) (bool, error) {
	g, err := fs.Glob(data.(fs.FS), condition)
	if err != nil {
		return false, err
	}
	return len(g) > 0, nil
}
