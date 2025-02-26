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

// LoadEvaluatorConfig provides config for making an eval.NewEvaluator with the default options and expression cache.
func LoadEvaluatorConfig() (*eval.Config, error) {
	cache, err := eval.NewFileCacheWithContent(exprCache, "")
	if err != nil {
		return nil, err
	}
	return &eval.Config{Cache: cache, EnvOptions: celfuncs.DefaultEnvOptions()}, nil
}

// LoadEmbeddedRulesets loads the rulesets embedded by what.ConfigData.
func LoadEmbeddedRulesets() (map[string]*rules.Ruleset, error) {
	var sets = make(map[string]*rules.Ruleset)
	if err := rules.ParseFiles(what.ConfigData, "config", sets); err != nil {
		return nil, err
	}
	return sets, nil
}
