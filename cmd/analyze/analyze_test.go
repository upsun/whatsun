package main

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"
	"what/internal/pm"

	"github.com/stretchr/testify/assert"

	"what"
	"what/analyzers/apps"
)

func TestAnalyze(t *testing.T) {
	testFs := fstest.MapFS{
		"composer.json":                    &fstest.MapFile{},
		"composer.lock":                    &fstest.MapFile{},
		".ignored":                         &fstest.MapFile{Mode: fs.ModeDir},
		".ignored/file":                    &fstest.MapFile{},
		"vendor":                           &fstest.MapFile{Mode: fs.ModeDir},
		"vendor/symfony":                   &fstest.MapFile{Mode: fs.ModeDir},
		"another-app/package-lock.json":    &fstest.MapFile{},
		"another-app/nested/composer.lock": &fstest.MapFile{},
		"some/very/deep/path/containing/a/composer.lock": &fstest.MapFile{},
		"configured-app/.platform.app.yaml":              &fstest.MapFile{},
	}

	resultChan := make(chan resultContext)
	analyze(context.TODO(), []what.Analyzer{&apps.Analyzer{
		MaxDepth: 2,
	}}, testFs, resultChan)

	r := <-resultChan
	assert.NoError(t, r.err)
	assert.Equal(t, r.Analyzer.String(), "apps")

	assert.EqualValues(t, apps.List{
		{Dir: ".", PackageManagers: pm.List{{PM: pm.Composer, Sources: []string{"composer.json", "composer.lock"}}}},
		{Dir: "another-app", PackageManagers: pm.List{{PM: pm.NPM, Sources: []string{"package-lock.json"}}}},
		{Dir: "configured-app"},
	}, r.Result.(apps.List))
}
