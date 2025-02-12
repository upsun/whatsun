// Package celfuncs provides functions for use in a Common Expression Language (CEL) environment.
package celfuncs

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// bytesStringReturnsStringErr returns a CEL function wrapping the given one with the same signature.
func bytesStringReturnsStringErr(name string, f func([]byte, string) (string, error)) cel.EnvOption {
	return cel.Function(
		name,
		cel.Overload(name,
			[]*cel.Type{cel.BytesType, cel.StringType},
			cel.StringType,
			cel.BinaryBinding(func(lhs, rhs ref.Val) ref.Val {
				s1, ok := lhs.(types.Bytes)
				if !ok {
					return types.NewErr("invalid argument 1")
				}
				s2, ok := rhs.(types.String)
				if !ok {
					return types.NewErr("invalid argument 2")
				}
				res, err := f(s1, string(s2))
				if err != nil {
					return types.NewErrFromString(err.Error())
				}
				return types.String(res)
			}),
		),
	)
}

// stringReturnsMapErr returns a CEL function wrapping the given one with the same signature.
func stringReturnsMapErr(name string, f func(s string) (map[string]string, error)) cel.EnvOption {
	return cel.Function(
		name,
		cel.Overload(name,
			[]*cel.Type{cel.StringType},
			cel.MapType(cel.StringType, cel.StringType),
			cel.UnaryBinding(func(arg ref.Val) ref.Val {
				str, ok := arg.(types.String)
				if !ok {
					return types.NewErr("invalid argument")
				}
				res, err := f(string(str))
				if err != nil {
					return types.NewErrFromString(err.Error())
				}
				return types.DefaultTypeAdapter.NativeToValue(res)
			}),
		),
	)
}
