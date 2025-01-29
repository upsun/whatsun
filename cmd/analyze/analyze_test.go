package main

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"what"
	"what/analysis"
)

func TestAnalyze(t *testing.T) {
	testFs := fstest.MapFS{
		"composer.lock":                 &fstest.MapFile{}, // ignored because 0 size
		"package-lock.json":             &fstest.MapFile{Data: []byte("{}")},
		".ignored":                      &fstest.MapFile{Mode: fs.ModeDir},
		".ignored/file":                 &fstest.MapFile{},
		"vendor":                        &fstest.MapFile{Mode: fs.ModeDir},
		"vendor/symfony":                &fstest.MapFile{Mode: fs.ModeDir},
		"another-app/package-lock.json": &fstest.MapFile{Data: []byte("{}")},
		"some/very/deep/path/containing/a/composer.lock": &fstest.MapFile{Data: []byte("{}")},
	}

	resultChan := make(chan resultContext)
	analyze(context.TODO(), []what.Analyzer{&analysis.ProjectAnalyzer{}}, testFs, ".", resultChan)

	r := <-resultChan
	assert.NoError(t, r.err)
	assert.Equal(t, r.Analyzer.GetName(), "project")
	assert.IsType(t, &analysis.Project{}, r.Result)
	paths := r.Result.(*analysis.Project).AppList.Paths
	assert.EqualValues(t, []string{"another-app", "."}, paths)
}
