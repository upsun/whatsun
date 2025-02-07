package main

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"
	"what/internal/rules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what"
	"what/internal/match"
)

func TestAnalyze(t *testing.T) {
	testFs := fstest.MapFS{
		// Definitely Composer.
		`composer.json`: &fstest.MapFile{Data: []byte(`{"require": {"symfony/framework-bundle": "^7"}}`)},
		`composer.lock`: &fstest.MapFile{Data: []byte(`{"packages": [{"name": "symfony/framework-bundle", "version": "7.2.3"}]}`)},

		// Ignored due to being a directory with the dot prefix.
		".ignored":      &fstest.MapFile{Mode: fs.ModeDir},
		".ignored/file": &fstest.MapFile{},

		// Potentially Composer or perhaps others.
		"vendor":         &fstest.MapFile{Mode: fs.ModeDir},
		"vendor/symfony": &fstest.MapFile{Mode: fs.ModeDir},

		// Definitely NPM.
		"another-app/package-lock.json": &fstest.MapFile{},

		// Ignored due to depth or nesting.
		"another-app/nested/composer.lock":          &fstest.MapFile{},
		"some/deep/path/containing/a/composer.json": &fstest.MapFile{},

		// Detected without having a package manager.
		"configured-app/.platform.app.yaml": &fstest.MapFile{},

		// Ambiguous: Bun, NPM, PNPM, or Yarn.
		"ambiguous/package.json": &fstest.MapFile{Data: []byte("{}")},
	}

	resultChan := make(chan resultContext)

	rulesAnalyzer, err := rules.NewAnalyzer()
	require.NoError(t, err)

	analyze(context.TODO(), []what.Analyzer{rulesAnalyzer}, testFs, resultChan)

	r := <-resultChan
	require.NoError(t, r.err)
	assert.Equal(t, r.Analyzer.String(), "rules")

	assert.EqualValues(t, rules.Results{
		"package_managers": {
			Directories: map[string][]match.Match{
				".": {{
					Result: "composer",
					Report: []rules.Report{{When: `file.exists("composer.json")`}},
					Sure:   true,
				}},
				"ambiguous": {
					{Result: "bun", Report: []rules.Report{{When: `file.exists("package.json")`}}},
					{Result: "npm", Report: []rules.Report{{When: `file.exists("package.json")`}}},
					{Result: "pnpm", Report: []rules.Report{{When: `file.exists("package.json")`}}},
					{Result: "yarn", Report: []rules.Report{{When: `file.exists("package.json")`}}},
				},
				"another-app": {{
					Result: "npm",
					Report: []rules.Report{{When: `file.exists("package-lock.json")`}},
					Sure:   true,
				}},
			},
		},
		"frameworks": {
			Directories: map[string][]match.Match{
				".": {{
					Result: "symfony",
					Report: []rules.Report{{
						When: `composer.requires("symfony/framework-bundle")`,
						With: map[string]string{"major_version": "7"},
					}},
					Sure: true,
				}},
			},
		},
	}, r.Result.(rules.Results))
}
