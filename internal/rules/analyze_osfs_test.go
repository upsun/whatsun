package rules_test

import (
	"context"
	"embed"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/internal/eval"
	"what/internal/eval/celfuncs"
	"what/internal/rules"
)

//go:embed testdata/rulesets
var testRulesetsDir embed.FS

// loadMockRulesets loads the rulesets embedded in the testdata/rulesets directory.
func loadMockRulesets() (map[string]*rules.Ruleset, error) {
	var sets = make(map[string]*rules.Ruleset)
	if err := rules.ParseFiles(testRulesetsDir, "testdata/rulesets", sets); err != nil {
		return nil, err
	}
	return sets, nil
}

// Test analysis on a real filesystem, but with mocked files and rulesets.
func TestAnalyze_OSFS_MockRules(t *testing.T) {
	rulesets, err := loadMockRulesets()
	require.NoError(t, err)
	cache, err := eval.NewFileCache("testdata/expr.cache")
	require.NoError(t, err)
	defer cache.Save()
	ev, err := eval.NewEvaluator(&eval.Config{Cache: cache, EnvOptions: celfuncs.DefaultEnvOptions()})
	require.NoError(t, err)

	analyzer := rules.NewAnalyzer(rulesets, ev, []string{"arg-ignored"})

	result, err := analyzer.Analyze(context.Background(), os.DirFS("testdata/mock-project"), ".")
	require.NoError(t, err)

	assert.EqualValues(t, rules.Results{
		"package_managers": {
			{Path: ".", Result: "npm", Sure: true, Rules: []string{"npm"}, Groups: []string{"js"}},
			{Path: "deep/1/2/3", Result: "npm", Sure: true, Rules: []string{"npm"}, Groups: []string{"js"}},
			{Path: "deep/1/2/python", Result: "pip", Sure: true, Rules: []string{"pip"}, Groups: []string{"python"}},
			{Path: "deep/1/2/python", Result: "poetry", Sure: true, Rules: []string{"poetry"}, Groups: []string{"python"}},
			{Path: "drupal", Result: "composer", Sure: true, Rules: []string{"composer"}, Groups: []string{"php"}},
			{Path: "symfony", Result: "composer", Sure: true, Rules: []string{"composer"}, Groups: []string{"php"}},
		},
	}, result)
}

// Benchmark analysis on a real filesystem, but with mocked files and rulesets.
func BenchmarkAnalyze_OSFS_MockRules(b *testing.B) {
	rulesets, err := loadMockRulesets()
	require.NoError(b, err)
	cache, err := eval.NewFileCache("testdata/expr.cache")
	require.NoError(b, err)
	ev, err := eval.NewEvaluator(&eval.Config{Cache: cache, EnvOptions: celfuncs.DefaultEnvOptions()})
	require.NoError(b, err)

	analyzer := rules.NewAnalyzer(rulesets, ev, []string{"arg-ignored"})
	require.NoError(b, err)

	fsys := os.DirFS("testdata/mock-project")

	ctx := context.Background()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err = analyzer.Analyze(ctx, fsys, ".")
		require.NoError(b, err)
	}
}
