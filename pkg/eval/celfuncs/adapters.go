package celfuncs

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// binaryFunction makes a CEL environment option for a function that takes 2 arguments.
func binaryFunction[ARG1 any, ARG2 any, R any](name string, argTypes []*cel.Type, returnType *cel.Type,
	f func(ARG1, ARG2) (R, error)) cel.EnvOption {
	return cel.Function(
		name,
		cel.Overload(name, argTypes, returnType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				res, err := f(lhs.Value().(ARG1), rhs.Value().(ARG2)) //nolint:errcheck
				if err != nil {
					return types.WrapErr(err)
				}
				return types.DefaultTypeAdapter.NativeToValue(res)
			}),
		),
	)
}

// unaryReceiverFunction makes a CEL environment option for a function that takes a receiver and 1 other argument.
func unaryReceiverFunction[REC any, ARG any, RET any](
	receiverName, functionName string,
	argTypes []*cel.Type,
	returnType *cel.Type,
	f func(REC, ARG) (RET, error),
) cel.EnvOption {
	overloadID := receiverName + "." + functionName

	return cel.Function(functionName,
		cel.MemberOverload(overloadID, argTypes, returnType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				res, err := f(lhs.Value().(REC), rhs.Value().(ARG)) //nolint:errcheck
				if err != nil {
					return types.WrapErr(err)
				}
				return types.DefaultTypeAdapter.NativeToValue(res)
			}),
		),
	)
}

// binaryReceiverFunction makes a CEL environment option for a function that takes a receiver and 2 other arguments.
func binaryReceiverFunction[REC any, ARG1 any, ARG2 any, RET any](
	receiverName, functionName string,
	argTypes []*cel.Type,
	returnType *cel.Type,
	f func(REC, ARG1, ARG2) (RET, error),
) cel.EnvOption {
	overloadID := receiverName + "." + functionName

	return cel.Function(functionName,
		cel.MemberOverload(overloadID, argTypes, returnType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				res, err := f(args[0].Value().(REC), args[1].Value().(ARG1), args[2].Value().(ARG2)) //nolint:errcheck
				if err != nil {
					return types.WrapErr(err)
				}
				return types.DefaultTypeAdapter.NativeToValue(res)
			}),
		),
	)
}
