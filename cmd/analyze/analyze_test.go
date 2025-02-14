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
		// No lockfile so generates an error getting the version.
		"ambiguous/package.json": &fstest.MapFile{Data: []byte(`{"dependencies":{"gatsby":"^5.14.1"}}`)},

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
	}, result["package_managers"].Paths["."])

	assert.EqualValues(t, []rules.Report{
		{Result: "bun", Rules: []string{"js-packages"}, Groups: []string{"js"}},
		{Result: "npm", Rules: []string{"js-packages"}, Groups: []string{"js"}},
		{Result: "pnpm", Rules: []string{"js-packages"}, Groups: []string{"js"}},
		{Result: "yarn", Rules: []string{"js-packages"}, Groups: []string{"js"}},
	}, result["package_managers"].Paths["ambiguous"])

	assert.EqualValues(t, []rules.Report{
		{Result: "gatsby", Sure: true, Rules: []string{"gatsby"}, With: map[string]rules.Metadata{
			"version": {Value: ""}, // When no version number is available.
		}, Groups: []string{"js"}},
	}, result["frameworks"].Paths["ambiguous"])

	assert.EqualValues(t, []rules.Report{
		{Result: "npm", Rules: []string{"npm-lockfile"}, Sure: true, Groups: []string{"js"}},
	}, result["package_managers"].Paths["another-app"])

	assert.EqualValues(t, []rules.Report{{
		Result: "symfony",
		Rules:  []string{"symfony-framework"},
		With:   map[string]rules.Metadata{"version": {Value: "7.2.3"}},
		Sure:   true,
		Groups: []string{"php"},
	}}, result["frameworks"].Paths["."])

	// A conflict will report an error without failing the whole ruleset.
	var conflictErr string
	var meteorMatches = make([]rules.Report, 0, len(result["package_managers"].Paths["meteor"])-1)
	for _, report := range result["package_managers"].Paths["meteor"] {
		if report.Error != "" {
			conflictErr = report.Error
		} else {
			meteorMatches = append(meteorMatches, report)
		}
	}
	assert.Contains(t, conflictErr, "conflict found in group js")

	assert.EqualValues(t, []rules.Report{
		{Result: "meteor", Rules: []string{"meteor"}, Sure: true, Groups: []string{"js"}},
		{Result: "npm", Rules: []string{"npm-lockfile"}, Sure: true, Groups: []string{"js"}},
	}, meteorMatches)
}
