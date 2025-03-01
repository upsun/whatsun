package what

import (
	"embed"

	"what/pkg/eval"
	"what/pkg/rules"
)

//go:embed expr.cache
var exprCache []byte

//go:embed config
var configData embed.FS

// LoadExpressionCache provides a default eval.Cache pre-warmed cache for the embedded expressions.
func LoadExpressionCache() (eval.Cache, error) {
	return eval.NewFileCacheWithContent(exprCache, "")
}

// LoadRulesets loads default rulesets.
func LoadRulesets() ([]rules.RulesetSpec, error) {
	return rules.LoadFromYAMLDir(configData, "config")
}
