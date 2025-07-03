package rules_test

import (
	_ "embed"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upsun/whatsun"
	"github.com/upsun/whatsun/pkg/rules"
)

var (
	//go:embed testdata/mock-django/pyproject.toml
	djangoPyProject []byte
	//go:embed testdata/mock-django/uv.lock
	djangoUvLock []byte

	//go:embed testdata/mock-blazor/BlazorApp.csproj
	blazorCsproj []byte
	//go:embed testdata/mock-blazor/packages.lock.json
	blazorLock []byte
)

var testFs = fstest.MapFS{
	".gitignore": &fstest.MapFile{Data: []byte("/git-ignored/\n" +
		"git-ignored-deep/\n" +
		"git-ignored-wildcard*\n",
	)},

	// Definitely Composer.
	`composer.json`: &fstest.MapFile{Data: []byte(`{"require": {"symfony/framework-bundle": "^7", "php": "^8.3"}}`)},
	`composer.lock`: &fstest.MapFile{Data: []byte(
		`{"packages": [{"name": "symfony/framework-bundle", "version": "7.2.3"}]}`)},

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

	"rake/Rakefile": &fstest.MapFile{},

	// Ambiguous: Bun, NPM, PNPM, or Yarn.
	// No lockfile so generates an error getting the version.
	"ambiguous/package.json": &fstest.MapFile{Data: []byte(`{"dependencies":{"gatsby":"^5.14.1"}}`)},

	// Eleventy: detected via glob.
	"eleventy/eleventy.config.ts": &fstest.MapFile{},

	// Meteor and NPM directory ("conflicting").
	"meteor/.meteor":           &fstest.MapFile{Mode: fs.ModeDir},
	"meteor/.meteor/packages":  &fstest.MapFile{Data: []byte("meteor-base")},
	"meteor/.meteor/versions":  &fstest.MapFile{Data: []byte("meteor-base@1.5.1")},
	"meteor/package-lock.json": &fstest.MapFile{Data: []byte("{}")},

	// Python using uv.lock.
	"python/pyproject.toml": &fstest.MapFile{Data: djangoPyProject},
	"python/uv.lock":        &fstest.MapFile{Data: djangoUvLock},

	// Blazor project.
	"blazor-app/BlazorApp.csproj":   &fstest.MapFile{Data: blazorCsproj},
	"blazor-app/packages.lock.json": &fstest.MapFile{Data: blazorLock},

	// Additional directories to increase time taken.
	"deep/1/2/3/4/5/composer.json":     &fstest.MapFile{Data: []byte("{}")},
	"deep/a/b/c/d/e/package.json":      &fstest.MapFile{Data: []byte("{}")},
	"deep/a/b/c/d/e/package-lock.json": &fstest.MapFile{Data: []byte("{}")},
}

func setupAnalyzerWithEmbeddedConfig(t require.TestingT, ignore []string) *rules.Analyzer {
	rulesets, err := whatsun.LoadRulesets()
	require.NoError(t, err)
	cache, err := whatsun.LoadExpressionCache()
	require.NoError(t, err)
	a, err := rules.NewAnalyzer(rulesets, &rules.AnalyzerConfig{
		CELExpressionCache: cache,
		IgnoreDirs:         ignore,
	})
	require.NoError(t, err)
	return a
}

// Test analysis on the test filesystem, but with real rulesets.
func TestAnalyze_TestFS_ActualRules(t *testing.T) {
	analyzer := setupAnalyzerWithEmbeddedConfig(t, []string{"arg-ignore"})

	reports, err := analyzer.Analyze(t.Context(), testFs, ".")
	require.NoError(t, err)

	assert.EqualValues(t, []rules.Report{
		// Build tool results.
		{Ruleset: "build_tools", Path: "configured-app", Result: "platformsh-app", Rules: []string{"platformsh-app"},
			With: map[string]rules.ReportValue{"name": {Value: "app"}}, Groups: []string{"cloud"}},
		{Ruleset: "build_tools", Path: "rake", Result: "rake", Rules: []string{"rake"}, Groups: []string{"ruby"}},

		// Framework results.
		{Ruleset: "frameworks", Path: ".", Result: "symfony", Rules: []string{"symfony-framework"},
			ReadFiles: []string{"compose.yaml"},
			With:      map[string]rules.ReportValue{"version": {Value: "7.2.3"}}, Groups: []string{"php", "symfony"}},
		{Ruleset: "frameworks", Path: "ambiguous", Result: "gatsby", Rules: []string{"gatsby"},
			With: map[string]rules.ReportValue{"version": {Value: ""}}, Groups: []string{"js"}},
		{Ruleset: "frameworks", Path: "blazor-app", Result: "blazor-wasm", Rules: []string{"blazor-wasm"},
			With: map[string]rules.ReportValue{"version": {Value: "8.0.0"}}, Groups: []string{"blazor", "dotnet"}},
		{Ruleset: "frameworks", Path: "eleventy", Result: "eleventy", Rules: []string{"eleventy"},
			With: map[string]rules.ReportValue{"version": {Value: ""}}, Groups: []string{"js", "static"}},
		{Ruleset: "frameworks", Path: "meteor", Result: "meteor.js", Rules: []string{"meteor.js"},
			With: map[string]rules.ReportValue{"version": {Value: "1.5.1"}}, Groups: []string{"js"}},
		{Ruleset: "frameworks", Path: "python", Result: "django", Rules: []string{"django"},
			With: map[string]rules.ReportValue{"version": {Value: "5.2.3"}}, Groups: []string{"django", "python"}},

		// Package manager results.
		{Ruleset: "package_managers", Path: ".", Result: "composer", Rules: []string{"composer"}, Groups: []string{"php"},
			ReadFiles: []string{"composer.json"},
			With:      map[string]rules.ReportValue{"php_version": {Value: "^8.3"}}},
		{Ruleset: "package_managers", Path: "ambiguous", Result: "bun", Maybe: true,
			ReadFiles: []string{"package.json"},
			Rules:     []string{"js-packages"}, Groups: []string{"js"}},
		{Ruleset: "package_managers", Path: "ambiguous", Result: "npm", Maybe: true,
			ReadFiles: []string{"package.json"},
			Rules:     []string{"js-packages"}, Groups: []string{"js"}},
		{Ruleset: "package_managers", Path: "ambiguous", Result: "pnpm", Maybe: true,
			ReadFiles: []string{"package.json"},
			Rules:     []string{"js-packages"}, Groups: []string{"js"}},
		{Ruleset: "package_managers", Path: "ambiguous", Result: "yarn", Maybe: true,
			ReadFiles: []string{"package.json"},
			Rules:     []string{"js-packages"}, Groups: []string{"js"}},
		{Ruleset: "package_managers", Path: "another-app", Result: "npm",
			Rules: []string{"npm-lockfile"}, Groups: []string{"js"}},
		{Ruleset: "package_managers", Path: "blazor-app", Result: "msbuild",
			Rules: []string{"msbuild"}, Groups: []string{"dotnet"}},
		{Ruleset: "package_managers", Path: "deep/1/2/3/4/5", Result: "composer",
			ReadFiles: []string{"composer.json"},
			Rules:     []string{"composer"}, Groups: []string{"php"},
			With: map[string]rules.ReportValue{"php_version": {Value: ""}}},
		{Ruleset: "package_managers", Path: "deep/a/b/c/d/e", Result: "npm",
			Rules: []string{"npm-lockfile"}, Groups: []string{"js"}},
		{Ruleset: "package_managers", Path: "meteor", Result: "meteor",
			Rules: []string{"meteor"}, Groups: []string{"js"}},
		{Ruleset: "package_managers", Path: "meteor", Result: "npm",
			Rules: []string{"npm-lockfile"}, Groups: []string{"js"}},
		{Ruleset: "package_managers", Path: "python", Result: "uv",
			Rules: []string{"uv"}, Groups: []string{"python"}},
	}, reports)
}

// Benchmark analysis on the test filesystem, but with real rulesets.
func BenchmarkAnalyze_TestFS_ActualRules(b *testing.B) {
	analyzer := setupAnalyzerWithEmbeddedConfig(b, []string{"arg-ignore"})

	ctx := b.Context()
	b.ReportAllocs()

	for b.Loop() {
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

	analyzer, err := rules.NewAnalyzer(rulesets, nil)
	require.NoError(t, err)

	result, err := analyzer.Analyze(t.Context(), fsys, ".")
	require.NoError(t, err)

	assert.EqualValues(t, []rules.Report{
		{Ruleset: "custom", Path: "bar", Result: "foo", Rules: []string{"foo-json"}},
		{Ruleset: "custom", Path: "deep/a/b/c", Result: "foo", Rules: []string{"foo-json"}},
		{Ruleset: "custom", Path: "foo", Result: "foo", Rules: []string{"foo-json"}},
	}, result)
}
