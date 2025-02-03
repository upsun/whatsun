// Package pm detects project managers in a code directory.
package pm

import (
	"errors"

	"github.com/google/cel-go/common/types"

	"what/internal/eval"
	"what/internal/match"
)

// Detect looks for evidence of package managers in a directory.
func Detect(ev *eval.Evaluator) ([]match.Match, error) {
	r, err := rules()
	if err != nil {
		return nil, err
	}

	return (&match.Matcher{Rules: r}).Match(func(condition string) (bool, error) {
		val, err := ev.Eval(condition)
		if err != nil {
			return false, err
		}
		switch val.(type) {
		case types.Bool:
			return bool(val.(types.Bool)), nil
		case types.String:
			return string(val.(types.String)) != "", nil
		case *types.Optional:
			return val.(*types.Optional).HasValue(), nil
		case types.Null:
			return false, nil
		}
		return false, errors.New("condition returns unexpected type")
	})
}
