package main

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"what"
	"what/analyzers/apps"
	"what/internal/heuristic"
)

func TestAnalyze(t *testing.T) {
	testFs := fstest.MapFS{
		// Definitely Composer.
		"composer.json": &fstest.MapFile{},
		"composer.lock": &fstest.MapFile{},

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
		"ambiguous/package.json": &fstest.MapFile{},
	}

	resultChan := make(chan resultContext)
	analyze(context.TODO(), []what.Analyzer{&apps.Analyzer{
		MaxDepth: 10,
	}}, testFs, resultChan)

	r := <-resultChan
	assert.NoError(t, r.err)
	assert.Equal(t, r.Analyzer.String(), "apps")

	assert.EqualValues(t, apps.List{
		{Dir: ".", PackageManagers: []heuristic.Finding{{Name: "composer", Sources: []string{"composer.json", "composer.lock"}}}},
		{Dir: "ambiguous", PackageManagers: []heuristic.Finding{
			{Name: "bun", Sources: []string{"package.json"}},
			{Name: "npm", Sources: []string{"package.json"}},
			{Name: "pnpm", Sources: []string{"package.json"}},
			{Name: "yarn", Sources: []string{"package.json"}},
		}},
		{Dir: "another-app", PackageManagers: []heuristic.Finding{{Name: "npm", Sources: []string{"package-lock.json"}}}},
		{Dir: "configured-app"},
	}, r.Result.(apps.List))
}
