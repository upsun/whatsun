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
	"what/internal/eval/celfuncs"
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
	ev, err := config.LoadEvaluator()
	require.NoError(t, err)

	return rules.NewAnalyzer(rulesets, ev, ignore)
}

// Test analysis on the test filesystem, but with real rulesets.
func TestAnalyze_TestFS_ActualRules(t *testing.T) {
	analyzer := setupAnalyzerWithEmbeddedConfig(t, []string{"arg-ignore"})

	result, err := analyzer.Analyze(context.Background(), testFs, ".")
	require.NoError(t, err)

	assert.EqualValues(t, rules.RulesetReports{
		"package_managers": {
			{
				Path:   ".",
				Result: "composer",
				Rules:  []string{"composer"},
				Groups: []string{"php"},
				With:   map[string]rules.ReportValue{"php_version": {Value: "^8.3"}},
			},
			{Path: "ambiguous", Result: "bun", Maybe: true, Rules: []string{"js-packages"}, Groups: []string{"js"}},
			{Path: "ambiguous", Result: "npm", Maybe: true, Rules: []string{"js-packages"}, Groups: []string{"js"}},
			{Path: "ambiguous", Result: "pnpm", Maybe: true, Rules: []string{"js-packages"}, Groups: []string{"js"}},
			{Path: "ambiguous", Result: "yarn", Maybe: true, Rules: []string{"js-packages"}, Groups: []string{"js"}},
			{Path: "another-app", Result: "npm", Rules: []string{"npm-lockfile"}, Groups: []string{"js"}},
			{Path: "meteor", Result: "meteor", Rules: []string{"meteor"}, Groups: []string{"js"}},
			{Path: "meteor", Result: "npm", Rules: []string{"npm-lockfile"}, Groups: []string{"js"}},
		},
		"frameworks": {
			{
				Path:   ".",
				Result: "symfony",
				Rules:  []string{"symfony-framework"},
				With:   map[string]rules.ReportValue{"version": {Value: "7.2.3"}},
				Groups: []string{"php", "symfony"},
			},
			{
				Path:   "ambiguous",
				Result: "gatsby",
				Rules:  []string{"gatsby"},
				With:   map[string]rules.ReportValue{"version": {Value: ""}},
				Groups: []string{"js"},
			},
			{
				Path:   "configured-app",
				Result: "platformsh-app",
				Rules:  []string{"platformsh-app"},
				With:   map[string]rules.ReportValue{"name": {Value: "app"}},
				Groups: []string{"cloud"},
			},
			{
				Path:   "eleventy",
				Result: "eleventy",
				Rules:  []string{"eleventy"},
				With:   map[string]rules.ReportValue{"version": {Value: ""}},
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

type customRuleset struct {
	name  string
	rules []rules.RuleSpec
}

func (c customRuleset) GetName() string            { return c.name }
func (c customRuleset) GetRules() []rules.RuleSpec { return c.rules }

type customRule struct {
	name      string
	condition string
	results   []string
}

func (c customRule) GetName() string      { return c.name }
func (c customRule) GetCondition() string { return c.condition }
func (c customRule) GetResults() []string { return c.results }

var _ rules.RulesetSpec = &customRuleset{}
var _ rules.RuleSpec = &customRule{}

// Test analysis with custom rules (not from YAML).
func TestAnalyze_CustomRules(t *testing.T) {
	fsys := fstest.MapFS{
		"foo/foo.json":        &fstest.MapFile{},
		"bar/foo.json":        &fstest.MapFile{},
		"deep/a/b/c/foo.json": &fstest.MapFile{},
		"not/bar.json":        &fstest.MapFile{},
	}

	rulesets := []rules.RulesetSpec{
		&customRuleset{name: "custom", rules: []rules.RuleSpec{
			&customRule{
				name:      "foo-json",
				condition: `fs.fileExists("foo.json")`,
				results:   []string{"foo"},
			},
		}},
	}
	ev, err := eval.NewEvaluator(&eval.Config{EnvOptions: celfuncs.DefaultEnvOptions()})
	require.NoError(t, err)

	analyzer := rules.NewAnalyzer(rulesets, ev, nil)

	result, err := analyzer.Analyze(context.Background(), fsys, ".")
	require.NoError(t, err)

	assert.EqualValues(t, rules.RulesetReports{
		"custom": {
			{Path: "bar", Result: "foo", Rules: []string{"foo-json"}},
			{Path: "deep/a/b/c", Result: "foo", Rules: []string{"foo-json"}},
			{Path: "foo", Result: "foo", Rules: []string{"foo-json"}},
		},
	}, result)
}
