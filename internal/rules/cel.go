package rules

import (
	"github.com/google/cel-go/cel"

	"what/internal/eval/celfuncs"
)

func DefaultEnvOptions() []cel.EnvOption {
	var celOptions []cel.EnvOption
	celOptions = append(celOptions, celfuncs.FilesystemVariable())
	celOptions = append(celOptions, celfuncs.AllFileFunctions()...)
	celOptions = append(celOptions, celfuncs.AllPackageManagerFunctions()...)

	return append(
		celOptions,
		celfuncs.JQ(),
		celfuncs.ParseVersion(),
	)
}
