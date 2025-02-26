package celfuncs

import "github.com/google/cel-go/cel"

// DefaultEnvOptions returns default options for creating a Common Expression Language (CEL) environment.
func DefaultEnvOptions() []cel.EnvOption {
	var celOptions []cel.EnvOption
	celOptions = append(celOptions, FilesystemVariable())
	celOptions = append(celOptions, AllFileOptions()...)
	celOptions = append(celOptions, AllPackageManagerFunctions()...)

	return append(
		celOptions,
		JQ(),
		YQ(),
		ParseVersion(),
	)
}
