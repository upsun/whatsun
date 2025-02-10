package rules

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"

	"what/internal/eval"
	"what/internal/eval/celfuncs"
)

func DefaultEnvOptions(fsys fs.FS, root *string) []cel.EnvOption {
	var celOptions []cel.EnvOption
	celOptions = append(celOptions, celfuncs.AllFileFunctions(&fsys, root)...)
	celOptions = append(celOptions, celfuncs.AllPackageManagerFunctions(&fsys, root)...)

	return append(
		celOptions,
		celfuncs.JSONQueryStringCELFunction(),
		celfuncs.VersionParse(),
	)
}

func evalFunc(ev *eval.Evaluator) func(string) (bool, error) {
	return func(condition string) (bool, error) {
		val, err := ev.Eval(condition)
		if err != nil {
			return false, err
		}

		asBool := val.ConvertToType(types.BoolType)
		if types.IsError(asBool) {
			return false, fmt.Errorf("%v", asBool)
		}

		return bool(asBool.(types.Bool)), nil
	}
}

func reportFunc(ev *eval.Evaluator) func(rules []*Rule) any {
	return func(rules []*Rule) any {
		var reports []Report
		for _, rule := range rules {
			rep := Report{Rule: rule.Name}
			if rep.Rule == "" && rule.When != "" {
				rep.Rule = "when: " + rule.When
			}
			if len(rule.With) == 0 {
				reports = append(reports, rep)
				continue
			}
			rep.With = make(map[string]string)
			for name, expr := range rule.With {
				val, err := ev.Eval(expr)
				if err != nil {
					rep.With[name] = fmt.Sprint("[ERROR] ", err.Error())
					continue
				}
				rep.With[name] = fmt.Sprint(val)
			}
			reports = append(reports, rep)
		}
		slices.SortFunc(reports, func(a, b Report) int {
			return strings.Compare(a.Rule, b.Rule)
		})
		return reports
	}
}
