package config

import (
	_ "embed"

	"what"
	"what/internal/eval"
	"what/internal/rules"
)

//go:embed expr.cache
var exprCache []byte

// LoadExpressionCache provides a default eval.Cache pre-warmed cache for the embedded expressions.
func LoadExpressionCache() (eval.Cache, error) {
	return eval.NewFileCacheWithContent(exprCache, "")
}

// LoadEmbeddedRulesets loads the rulesets embedded by what.ConfigData.
func LoadEmbeddedRulesets() ([]rules.RulesetSpec, error) {
	return rules.LoadFromYAMLDir(what.ConfigData, "config")
}
