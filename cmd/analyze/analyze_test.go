package main

import (
	"context"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"what/internal/pm"

	"github.com/stretchr/testify/assert"

	"what"
	"what/analyzers/apps"
)

func TestAnalyze(t *testing.T) {
	testFs := fstest.MapFS{
		"composer.lock":                 &fstest.MapFile{},
		".ignored":                      &fstest.MapFile{Mode: fs.ModeDir},
		".ignored/file":                 &fstest.MapFile{},
		"vendor":                        &fstest.MapFile{Mode: fs.ModeDir},
		"vendor/symfony":                &fstest.MapFile{Mode: fs.ModeDir},
		"another-app/package-lock.json": &fstest.MapFile{},
		"some/very/deep/path/containing/a/composer.lock": &fstest.MapFile{},
		"configured-app/.platform.app.yaml":              &fstest.MapFile{},
	}

	resultChan := make(chan resultContext)
	analyze(context.TODO(), []what.Analyzer{apps.New()}, testFs, resultChan)

	r := <-resultChan
	assert.NoError(t, r.err)
	assert.Equal(t, r.Analyzer.GetName(), "apps")
	assert.IsType(t, apps.List{}, r.Result)

	list := r.Result.(apps.List)
	slices.SortFunc(list, func(a, b apps.App) int {
		return strings.Compare(a.Dir, b.Dir)
	})
	assert.EqualValues(t, apps.List{
		{Dir: ".", PackageManagers: pm.List{{Type: pm.Composer, DetectedVia: "composer.lock"}}, DetectedVia: "package_manager"},
		{Dir: "another-app", PackageManagers: pm.List{{Type: pm.NPM, DetectedVia: "package-lock.json"}}, DetectedVia: "package_manager"},
		{Dir: "configured-app", DetectedVia: ".platform.app.yaml"},
	}, list)
}
