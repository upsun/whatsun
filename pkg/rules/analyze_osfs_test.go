package rules_test

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upsun/whatsun/pkg/eval"
	"github.com/upsun/whatsun/pkg/rules"
)

//go:embed testdata/rulesets
var testRulesetsDir embed.FS

// Test analysis on a real filesystem, but with mocked files and rulesets.
func TestAnalyze_OSFS_MockRules(t *testing.T) {
	rulesets, err := rules.LoadFromYAMLDir(testRulesetsDir, "testdata/rulesets")
	require.NoError(t, err)
	cache, err := eval.NewFileCache("testdata/expr.cache")
	require.NoError(t, err)
	defer cache.Save() //nolint:errcheck

	analyzer, err := rules.NewAnalyzer(rulesets, &rules.AnalyzerConfig{
		CELExpressionCache: cache,
		IgnoreDirs:         []string{"arg-ignored"},
	})
	require.NoError(t, err)

	result, err := analyzer.Analyze(t.Context(), os.DirFS("testdata/mock-project"), ".")
	require.NoError(t, err)

	rs := "package_managers"
	assert.EqualValues(t, []rules.Report{
		{Path: ".", Result: "npm", Ruleset: rs, Rules: []string{"npm"}, Groups: []string{"js"}},
		{Path: "deep/1/2/3", Result: "npm", Ruleset: rs, Rules: []string{"npm"}, Groups: []string{"js"}},
		{Path: "deep/1/2/python", Result: "pip", Ruleset: rs, Rules: []string{"pip"}, Groups: []string{"python"}},
		{Path: "deep/1/2/python", Result: "poetry", Ruleset: rs, Rules: []string{"poetry"}, Groups: []string{"python"}},
		{Path: "drupal", Result: "composer", Ruleset: rs, Rules: []string{"composer"}, Groups: []string{"php"}},
		{Path: "symfony", Result: "composer", Ruleset: rs, Rules: []string{"composer"}, Groups: []string{"php"}},
	}, result)
}

// Test global gitignore support with real filesystem
func TestAnalyze_OSFS_GlobalGitignore(t *testing.T) {
	// Create temporary directory for mock home
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")

	require.NoError(t, os.MkdirAll(homeDir, 0755))
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "global-ignored"), 0755))

	// Create global gitignore file
	globalGitignore := filepath.Join(homeDir, ".gitignore")
	require.NoError(t, os.WriteFile(globalGitignore, []byte("global-ignored/\n"), 0600))

	// Create a project file that would be detected if not ignored
	composerJSON := filepath.Join(projectDir, "global-ignored", "composer.json")
	require.NoError(t, os.WriteFile(composerJSON, []byte(`{"require": {"php": "^8.0"}}`), 0600))

	// Set HOME to our temporary directory
	t.Setenv("HOME", homeDir)

	// Set up analyzer
	rulesets, err := rules.LoadFromYAMLDir(testRulesetsDir, "testdata/rulesets")
	require.NoError(t, err)
	cache, err := eval.NewFileCache("testdata/expr.cache")
	require.NoError(t, err)
	defer cache.Save() //nolint:errcheck

	analyzer, err := rules.NewAnalyzer(rulesets, &rules.AnalyzerConfig{
		CELExpressionCache: cache,
	})
	require.NoError(t, err)

	// Analyze the project
	reports, err := analyzer.Analyze(t.Context(), os.DirFS(projectDir), ".")
	require.NoError(t, err)

	// Should be empty because global-ignored/ should be ignored
	assert.Empty(t, reports)
}

// Benchmark analysis on a real filesystem, but with mocked files and rulesets.
func BenchmarkAnalyze_OSFS_MockRules(b *testing.B) {
	rulesets, err := rules.LoadFromYAMLDir(testRulesetsDir, "testdata/rulesets")
	require.NoError(b, err)
	cache, err := eval.NewFileCache("testdata/expr.cache")
	require.NoError(b, err)

	analyzer, err := rules.NewAnalyzer(rulesets, &rules.AnalyzerConfig{
		CELExpressionCache: cache,
		IgnoreDirs:         []string{"arg-ignored"},
	})
	require.NoError(b, err)

	fsys := os.DirFS("testdata/mock-project")

	ctx := b.Context()

	b.ReportAllocs()

	for b.Loop() {
		_, err = analyzer.Analyze(ctx, fsys, ".")
		require.NoError(b, err)
	}
}
