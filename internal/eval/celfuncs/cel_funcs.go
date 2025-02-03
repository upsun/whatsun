// Package celfuncs provides functions for use in a Common Expression Language (CEL) environment.
package celfuncs

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// stringReturnsBoolErr returns a CEL function wrapping the given one with the same signature.
func stringReturnsBoolErr(name string, f func(s string) (bool, error)) cel.EnvOption {
	return cel.Function(
		name,
		cel.Overload(name,
			[]*cel.Type{cel.StringType},
			cel.BoolType,
			cel.UnaryBinding(func(arg ref.Val) ref.Val {
				if str, ok := arg.(types.String); ok {
					res, err := f(string(str))
					if err != nil {
						return types.NewErrFromString(err.Error())
					}
					return types.Bool(res)
				}
				return types.NewErr("invalid argument")
			}),
		),
	)
}

// stringStringReturnsBoolErr returns a CEL function wrapping the given one with the same signature.
func stringStringReturnsBoolErr(name string, f func(s1, s2 string) (bool, error)) cel.EnvOption {
	return cel.Function(
		name,
		cel.Overload(name,
			[]*cel.Type{cel.StringType, cel.StringType},
			cel.BoolType,
			cel.BinaryBinding(func(lhs, rhs ref.Val) ref.Val {
				s1, ok := lhs.(types.String)
				if !ok {
					return types.NewErr("invalid argument 1")
				}
				s2, ok := rhs.(types.String)
				if !ok {
					return types.NewErr("invalid argument 2")
				}
				res, err := f(string(s1), string(s2))
				if err != nil {
					return types.NewErrFromString(err.Error())
				}
				return types.Bool(res)
			}),
		),
	)
}

// stringReturnsBytesErr returns a CEL function wrapping the given one with the same signature.
func stringReturnsBytesErr(name string, f func(s string) ([]byte, error)) cel.EnvOption {
	return cel.Function(
		name,
		cel.Overload(name,
			[]*cel.Type{cel.StringType},
			cel.BytesType,
			cel.UnaryBinding(func(arg ref.Val) ref.Val {
				if str, ok := arg.(types.String); ok {
					res, err := f(string(str))
					if err != nil {
						return types.NewErrFromString(err.Error())
					}
					return types.Bytes(res)
				}
				return types.NewErr("invalid argument")
			}),
		),
	)
}

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

// stringReturnsStringErr returns a CEL function wrapping the given one with the same signature.
func stringReturnsStringErr(name string, f func(string) (string, error)) cel.EnvOption {
	return cel.Function(
		name,
		cel.Overload(name,
			[]*cel.Type{cel.StringType},
			cel.StringType,
			cel.UnaryBinding(func(arg ref.Val) ref.Val {
				s, ok := arg.(types.String)
				if !ok {
					return types.NewErr("invalid argument")
				}
				res, err := f(string(s))
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
