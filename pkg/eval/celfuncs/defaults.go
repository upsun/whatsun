// Package celfuncs provides functions for use in a Common Expression Language (CEL) environment.
package celfuncs

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
)

// DefaultEnvOptions returns default options for creating a Common Expression Language (CEL) environment.
func DefaultEnvOptions() []cel.EnvOption {
	return append(CustomEnvOptions(), ext.Lists(), ext.Strings(), ext.NativeTypes())
}

// CustomEnvOptions returns the customized CEL options.
func CustomEnvOptions() []cel.EnvOption {
	var celOptions []cel.EnvOption
	celOptions = append(celOptions, FilesystemVariables()...)
	celOptions = append(celOptions, AllFileOptions()...)
	celOptions = append(celOptions, AllPackageManagerFunctions()...)
	return append(celOptions,
		JQ(),
		YQ(),
	)
}
