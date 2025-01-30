package what

import (
	"context"
	"errors"
	"io/fs"
)

type Analyzer interface {
	String() string // Return the name of the analyzer
	Analyze(context.Context, fs.FS) (Result, error)
}

var ErrNotApplicable = errors.New("not applicable")

type Result interface {
	String() string
}
