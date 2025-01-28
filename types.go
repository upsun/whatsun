package what

import (
	"context"
	"errors"
	"io/fs"
)

type Analyzer interface {
	GetName() string
	Analyze(context.Context, fs.FS, string) (Result, error)
}

var ErrNotApplicable = errors.New("not applicable")

type Result interface {
	GetSummary() string
}
