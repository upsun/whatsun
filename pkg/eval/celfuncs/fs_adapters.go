package celfuncs

import "github.com/google/cel-go/cel"

// fsUnaryFunction returns a CEL environment option for a receiver method on the "fs" variable that takes 1 argument.
func fsUnaryFunction[ARG any, RET any](name string, argType, returnType *cel.Type,
	f func(filesystemWrapper, ARG) (RET, error)) cel.EnvOption {
	return unaryReceiverFunction(fsVariable, name, []*cel.Type{cel.DynType, argType}, returnType, f)
}

// fsBinaryFunction returns a CEL environment option for a receiver method on the "fs" variable that takes 2 arguments.
func fsBinaryFunction[A1 any, A2 any, RET any](name string, argTypes []*cel.Type, returnType *cel.Type,
	f func(filesystemWrapper, A1, A2) (RET, error)) cel.EnvOption {
	return binaryReceiverFunction(fsVariable, name, append([]*cel.Type{cel.DynType}, argTypes...), returnType, f)
}
