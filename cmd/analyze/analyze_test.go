package main

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestAnalyze(t *testing.T) {
	testFs := fstest.MapFS{
		"composer.json":  &fstest.MapFile{},
		".ignored":       &fstest.MapFile{Mode: fs.ModeDir},
		".ignored/file":  &fstest.MapFile{},
		"vendor":         &fstest.MapFile{Mode: fs.ModeDir},
		"vendor/symfony": &fstest.MapFile{Mode: fs.ModeDir},
	}

	resultChan := make(chan resultContext)
	err := analyze(context.TODO(), testFs, ".", resultChan)
	assert.NoError(t, err)

	r := <-resultChan
	assert.Equal(t, r.Analyzer.Name(), "apps")
	assert.Len(t, r.results, 1)
	assert.Equal(t, ".", r.results[0].Payload)
	assert.Equal(t, "composer.json", r.results[0].Reason)
}
