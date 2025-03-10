package celfuncs

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// binaryFunction turns a native Go function (a binary one with 2 arguments) into a CEL environment option.
func binaryFunction[ARG1 any, ARG2 any, R any](name string, argTypes []*cel.Type, returnType *cel.Type, f func(ARG1, ARG2) (R, error)) cel.EnvOption {
	return cel.Function(
		name,
		cel.Overload(name,
			argTypes,
			returnType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				res, err := f(lhs.Value().(ARG1), rhs.Value().(ARG2))
				if err != nil {
					return types.WrapErr(err)
				}
				return types.DefaultTypeAdapter.NativeToValue(res)
			}),
		),
	)
}

// unaryReceiverFunction turns a native Go function (taking 2 arguments: a receiver and something else) into a CEL environment option.
func unaryReceiverFunction[REC any, ARG any, RET any](receiverName, functionName string, argTypes []*cel.Type, returnType *cel.Type, f func(REC, ARG) (RET, error)) cel.EnvOption {
	overloadID := receiverName + "." + functionName

	return cel.Function(functionName,
		cel.MemberOverload(
			overloadID,
			argTypes,
			returnType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				res, err := f(lhs.Value().(REC), rhs.Value().(ARG))
				if err != nil {
					return types.WrapErr(err)
				}
				return types.DefaultTypeAdapter.NativeToValue(res)
			}),
		),
	)
}

// binaryReceiverFunction turns a native Go function (taking 3 arguments: a receiver and two others) into a CEL environment option.
func binaryReceiverFunction[REC any, ARG1 any, ARG2 any, RET any](receiverName, functionName string, argTypes []*cel.Type, returnType *cel.Type, f func(REC, ARG1, ARG2) (RET, error)) cel.EnvOption {
	overloadID := receiverName + "." + functionName

	return cel.Function(functionName,
		cel.MemberOverload(
			overloadID,
			argTypes,
			returnType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				res, err := f(args[0].Value().(REC), args[1].Value().(ARG1), args[2].Value().(ARG2))
				if err != nil {
					return types.WrapErr(err)
				}
				return types.DefaultTypeAdapter.NativeToValue(res)
			}),
		),
	)
}
