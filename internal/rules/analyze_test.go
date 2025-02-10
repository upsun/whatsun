package rules_test

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
		".gitignore": &fstest.MapFile{Data: []byte("/git-ignored/\n" +
			"git-ignored-deep/\n" +
			"git-ignored-wildcard*\n",
		)},

		// Definitely Composer.
		`composer.json`: &fstest.MapFile{Data: []byte(`{"require": {"symfony/framework-bundle": "^7"}}`)},
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

	rulesAnalyzer, err := rules.NewAnalyzer([]string{"arg-ignored"})
	require.NoError(t, err)

	result, err := rulesAnalyzer.Analyze(context.Background(), testFs, ".")
	require.NoError(t, err)

	expectIgnored := []string{
		"git-ignored",
		"node_modules",
		"a/b/git-ignored-deep",
		"a/b/git-ignored-wildcard-example",
		"x/y/ignore-subdir",
		"arg-ignored",
	}
	for _, v := range expectIgnored {
		if _, ok := result["package_managers"].Paths[v]; ok {
			t.Error("Result found for path but it should be ignored:", v)
		}
	}

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
		Groups: []string{"php", "symfony"},
	}}, result["frameworks"].Paths["."])

	assert.EqualValues(t, []rules.Report{
		{Result: "meteor", Rules: []string{"meteor"}, Sure: true, Groups: []string{"js"}},
		{Result: "npm", Rules: []string{"npm-lockfile"}, Sure: true, Groups: []string{"js"}},
	}, result["package_managers"].Paths["meteor"])
}
