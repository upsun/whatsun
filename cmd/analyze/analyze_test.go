package main

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/internal/rules"
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
		"some/deep/path/containing/a/composer.json": &fstest.MapFile{Data: []byte("{}")},

		// Detected without having a package manager.
		"configured-app/.platform.app.yaml": &fstest.MapFile{},

		// Ambiguous: Bun, NPM, PNPM, or Yarn.
		"ambiguous/package.json": &fstest.MapFile{Data: []byte("{}")},

		// Meteor and NPM directory ("conflicting").
		"meteor/.meteor":           &fstest.MapFile{Mode: fs.ModeDir},
		"meteor/.meteor/packages":  &fstest.MapFile{},
		"meteor/.meteor/versions":  &fstest.MapFile{},
		"meteor/package-lock.json": &fstest.MapFile{},
	}

	rulesAnalyzer, err := rules.NewAnalyzer()
	require.NoError(t, err)

	result, err := rulesAnalyzer.Analyze(context.Background(), testFs, ".")
	require.NoError(t, err)

	assert.EqualValues(t, []rules.Report{{
		Result: "composer", Rules: []string{"composer"}, Sure: true},
	}, result["package_managers"].Directories["."])

	assert.EqualValues(t, []rules.Report{
		{Result: "bun", Rules: []string{"js-packages"}},
		{Result: "npm", Rules: []string{"js-packages"}},
		{Result: "pnpm", Rules: []string{"js-packages"}},
		{Result: "yarn", Rules: []string{"js-packages"}},
	}, result["package_managers"].Directories["ambiguous"])

	assert.EqualValues(t, []rules.Report{
		{Result: "npm", Rules: []string{"npm-lockfile"}, Sure: true},
	}, result["package_managers"].Directories["another-app"])

	assert.EqualValues(t, []rules.Report{{
		Result: "symfony",
		Rules:  []string{"symfony-framework"},
		With:   map[string]string{"major_version": "7"},
		Sure:   true,
	}}, result["frameworks"].Directories["."])

	// A conflict will report an error without failing the whole ruleset.
	var conflictErr error
	var meteorMatches = make([]rules.Report, 0, len(result["package_managers"].Directories["meteor"])-1)
	for _, report := range result["package_managers"].Directories["meteor"] {
		if report.Err != nil {
			conflictErr = report.Err
		} else {
			meteorMatches = append(meteorMatches, report)
		}
	}
	assert.ErrorContains(t, conflictErr, "conflict found in group js")
	assert.EqualValues(t, []rules.Report{
		{Result: "meteor", Rules: []string{"meteor"}, Sure: true},
		{Result: "npm", Rules: []string{"npm-lockfile"}, Sure: true},
	}, meteorMatches)
}
