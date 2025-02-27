package config

import (
	_ "embed"

	"what"
	"what/internal/eval"
	"what/internal/eval/celfuncs"
	"what/internal/rules"
)

//go:embed expr.cache
var exprCache []byte

// LoadEvaluator provides a default eval.Evaluator with a pre-warmed cache for the embedded expressions.
func LoadEvaluator() (*eval.Evaluator, error) {
	cache, err := eval.NewFileCacheWithContent(exprCache, "")
	if err != nil {
		return nil, err
	}
	return eval.NewEvaluator(&eval.Config{Cache: cache, EnvOptions: celfuncs.DefaultEnvOptions()})
}

// LoadEmbeddedRulesets loads the rulesets embedded by what.ConfigData.
func LoadEmbeddedRulesets() ([]rules.RulesetSpec, error) {
	return rules.LoadFromYAMLDir(what.ConfigData, "config")
}
