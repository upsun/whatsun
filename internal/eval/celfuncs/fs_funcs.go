package celfuncs

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func fsStringReturnsBoolErr(name string, f func(filesystemWrapper, string) (bool, error)) cel.EnvOption {
	return cel.Function(name,
		cel.MemberOverload(
			fsVariable+"."+name,
			[]*cel.Type{cel.DynType, cel.StringType},
			cel.BoolType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				fsWrapper, ok := lhs.Value().(filesystemWrapper)
				if !ok {
					return types.NewErr("invalid receiver type %T, expected filesystemWrapper", lhs.Value())
				}

				str, ok := rhs.Value().(string)
				if !ok {
					return types.NewErr("invalid argument type %T, expected string", rhs.Value())
				}

				res, err := f(fsWrapper, str)
				if err != nil {
					return types.WrapErr(err)
				}
				return types.Bool(res)
			}),
		),
	)
}

func fsStringReturnsBytesErr(name string, f func(filesystemWrapper, string) ([]byte, error)) cel.EnvOption {
	return cel.Function(name,
		cel.MemberOverload(
			fsVariable+"."+name,
			[]*cel.Type{cel.DynType, cel.StringType},
			cel.BytesType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				fsWrapper, ok := lhs.Value().(filesystemWrapper)
				if !ok {
					return types.NewErr("invalid receiver type %T, expected filesystemWrapper", lhs.Value())
				}

				str, ok := rhs.Value().(string)
				if !ok {
					return types.NewErr("invalid argument type %T, expected string", rhs.Value())
				}

				res, err := f(fsWrapper, str)
				if err != nil {
					return types.WrapErr(err)
				}
				return types.Bytes(res)
			}),
		),
	)
}

func fsStringReturnsListErr(name string, f func(filesystemWrapper, string) ([]string, error)) cel.EnvOption {
	return cel.Function(name,
		cel.MemberOverload(
			fsVariable+"."+name,
			[]*cel.Type{cel.DynType, cel.StringType},
			cel.ListType(cel.StringType),
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				fsWrapper, ok := lhs.Value().(filesystemWrapper)
				if !ok {
					return types.NewErr("invalid receiver type %T, expected filesystemWrapper", lhs.Value())
				}

				str, ok := rhs.Value().(string)
				if !ok {
					return types.NewErr("invalid argument type %T, expected string", rhs.Value())
				}

				res, err := f(fsWrapper, str)
				if err != nil {
					return types.WrapErr(err)
				}

				return types.DefaultTypeAdapter.NativeToValue(res)
			}),
		),
	)
}

func fsStringStringReturnsStringErr(name string, f func(filesystemWrapper, string, string) (string, error)) cel.EnvOption {
	return cel.Function(name,
		cel.MemberOverload(
			fsVariable+"."+name,
			[]*cel.Type{cel.DynType, cel.StringType, cel.StringType},
			cel.StringType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("%s() requires exactly 2 arguments", name)
				}

				fsWrapper, ok := args[0].Value().(filesystemWrapper)
				if !ok {
					return types.NewErr("invalid receiver type %T, expected filesystemWrapper", args[0])
				}

				str1, ok := args[1].Value().(string)
				if !ok {
					return types.NewErr("invalid argument 1 of type %T, expected string", args[1].Value())
				}

				str2, ok := args[2].Value().(string)
				if !ok {
					return types.NewErr("invalid argument 2 of type %T, expected string", args[2].Value())
				}

				res, err := f(fsWrapper, str1, str2)
				if err != nil {
					return types.WrapErr(err)
				}
				return types.String(res)
			}),
		),
	)
}

func fsStringStringReturnsBoolErr(name string, f func(filesystemWrapper, string, string) (bool, error)) cel.EnvOption {
	return cel.Function(name,
		cel.MemberOverload(
			fsVariable+"."+name,
			[]*cel.Type{cel.DynType, cel.StringType, cel.StringType},
			cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("%s() requires exactly 2 arguments", name)
				}

				fsWrapper, ok := args[0].Value().(filesystemWrapper)
				if !ok {
					return types.NewErr("invalid receiver type %T, expected filesystemWrapper", args[0])
				}

				str1, ok := args[1].Value().(string)
				if !ok {
					return types.NewErr("invalid argument 1 of type %T, expected string", args[1].Value())
				}

				str2, ok := args[2].Value().(string)
				if !ok {
					return types.NewErr("invalid argument 2 of type %T, expected string", args[2].Value())
				}

				res, err := f(fsWrapper, str1, str2)
				if err != nil {
					return types.WrapErr(err)
				}
				return types.Bool(res)
			}),
		),
	)
}
