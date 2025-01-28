package what

import (
	"context"
	"errors"
	"io/fs"
)

type Analyzer interface {
	Name() string
	Analyze(context.Context, fs.FS, string) ([]Result, error)
}

var (
	ErrNotApplicable = errors.New("not applicable")
)

type Result struct {
	Payload any
	Reason  any
}
