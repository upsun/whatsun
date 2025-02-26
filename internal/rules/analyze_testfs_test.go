package rules_test

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/internal/config"
	"what/internal/eval"
	"what/internal/rules"
)

var testFs = fstest.MapFS{
	".gitignore": &fstest.MapFile{Data: []byte("/git-ignored/\n" +
		"git-ignored-deep/\n" +
		"git-ignored-wildcard*\n",
	)},

	// Definitely Composer.
	`composer.json`: &fstest.MapFile{Data: []byte(`{"require": {"symfony/framework-bundle": "^7", "php": "^8.3"}}`)},
	`composer.lock`: &fstest.MapFile{Data: []byte(`{"packages": [{"name": "symfony/framework-bundle", "version": "7.2.3"}]}`)},

	// Ignored due to .gitignore.
	"git-ignored/composer.json":                      &fstest.MapFile{Data: []byte("{}")},
	"a/b/git-ignored-deep/composer.json":             &fstest.MapFile{Data: []byte("{}")},
	"a/b/git-ignored-wildcard-example/composer.json": &fstest.MapFile{Data: []byte("{}")},

	// Ignored due to .gitignore in a subdirectory.
	"x/y/.gitignore":                  &fstest.MapFile{Data: []byte("ignore-subdir/")},
	"x/y/ignore-subdir/composer.json": &fstest.MapFile{Data: []byte("{}")},

	// Ignored due to default ignores.
	"node_modules/composer.lock": &fstest.MapFile{Data: []byte("{}")},

	// Ignored due to argument ignores.
	"arg-ignored/composer.lock": &fstest.MapFile{Data: []byte("{}")},

	// Potentially Composer or perhaps others.
	"vendor":         &fstest.MapFile{Mode: fs.ModeDir},
	"vendor/symfony": &fstest.MapFile{Mode: fs.ModeDir},

	// Definitely NPM.
	"another-app/package.json":      &fstest.MapFile{Data: []byte("{}")},
	"another-app/package-lock.json": &fstest.MapFile{Data: []byte("{}")},

	// Detected without having a package manager.
	"configured-app/.platform.app.yaml": &fstest.MapFile{Data: []byte("name: app")},

	// Ambiguous: Bun, NPM, PNPM, or Yarn.
	// No lockfile so generates an error getting the version.
	"ambiguous/package.json": &fstest.MapFile{Data: []byte(`{"dependencies":{"gatsby":"^5.14.1"}}`)},

	// Eleventy: detected via glob.
	"eleventy/eleventy.config.ts": &fstest.MapFile{},

	// Meteor and NPM directory ("conflicting").
	"meteor/.meteor":           &fstest.MapFile{Mode: fs.ModeDir},
	"meteor/.meteor/packages":  &fstest.MapFile{},
	"meteor/.meteor/versions":  &fstest.MapFile{},
	"meteor/package-lock.json": &fstest.MapFile{},
}

func setupAnalyzerWithEmbeddedConfig(t require.TestingT, ignore []string) *rules.Analyzer {
	rulesets, err := config.LoadEmbeddedRulesets()
	require.NoError(t, err)
	evalConfig, err := config.LoadEvaluatorConfig()
	require.NoError(t, err)
	ev, err := eval.NewEvaluator(evalConfig)
	require.NoError(t, err)

	return rules.NewAnalyzer(rulesets, ev, ignore)
}

// Test analysis on the test filesystem, but with real rulesets.
func TestAnalyze_TestFS_ActualRules(t *testing.T) {
	analyzer := setupAnalyzerWithEmbeddedConfig(t, []string{"arg-ignore"})

	result, err := analyzer.Analyze(context.Background(), testFs, ".")
	require.NoError(t, err)

	assert.EqualValues(t, rules.Results{
		"package_managers": {
			{
				Path:   ".",
				Result: "composer",
				Sure:   true,
				Rules:  []string{"composer"},
				Groups: []string{"php"},
				With:   map[string]rules.Metadata{"php_version": {Value: "^8.3"}},
			},
			{Path: "ambiguous", Result: "bun", Rules: []string{"js-packages"}, Groups: []string{"js"}},
			{Path: "ambiguous", Result: "npm", Rules: []string{"js-packages"}, Groups: []string{"js"}},
			{Path: "ambiguous", Result: "pnpm", Rules: []string{"js-packages"}, Groups: []string{"js"}},
			{Path: "ambiguous", Result: "yarn", Rules: []string{"js-packages"}, Groups: []string{"js"}},
			{Path: "another-app", Result: "npm", Rules: []string{"npm-lockfile"}, Sure: true, Groups: []string{"js"}},
			{Path: "meteor", Result: "meteor", Rules: []string{"meteor"}, Sure: true, Groups: []string{"js"}},
			{Path: "meteor", Result: "npm", Rules: []string{"npm-lockfile"}, Sure: true, Groups: []string{"js"}},
		},
		"frameworks": {
			{
				Path:   ".",
				Result: "symfony",
				Rules:  []string{"symfony-framework"},
				With:   map[string]rules.Metadata{"version": {Value: "7.2.3"}},
				Sure:   true,
				Groups: []string{"php", "symfony"},
			},
			{
				Path:   "ambiguous",
				Result: "gatsby",
				Rules:  []string{"gatsby"},
				With:   map[string]rules.Metadata{"version": {Value: ""}},
				Sure:   true,
				Groups: []string{"js"},
			},
			{
				Path:   "configured-app",
				Result: "platformsh-app",
				Rules:  []string{"platformsh-app"},
				With:   map[string]rules.Metadata{"name": {Value: "app"}},
				Sure:   true,
				Groups: []string{"cloud"},
			},
			{
				Path:   "eleventy",
				Result: "eleventy",
				Rules:  []string{"eleventy"},
				With:   map[string]rules.Metadata{"version": {Value: ""}},
				Sure:   true,
				Groups: []string{"js", "static"},
			},
		},
	}, result)
}

// Benchmark analysis on the test filesystem, but with real rulesets.
func BenchmarkAnalyze_TestFS_ActualRules(b *testing.B) {
	analyzer := setupAnalyzerWithEmbeddedConfig(b, []string{"arg-ignore"})

	ctx := context.Background()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := analyzer.Analyze(ctx, testFs, ".")
		require.NoError(b, err)
	}
}
