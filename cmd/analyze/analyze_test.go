package main

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyze(t *testing.T) {
	testFs := fstest.MapFS{
		"composer.lock":     &fstest.MapFile{}, // ignored because 0 size
		"package-lock.json": &fstest.MapFile{Data: []byte("{}")},
		".ignored":          &fstest.MapFile{Mode: fs.ModeDir},
		".ignored/file":     &fstest.MapFile{},
		"vendor":            &fstest.MapFile{Mode: fs.ModeDir},
		"vendor/symfony":    &fstest.MapFile{Mode: fs.ModeDir},
		"some/very/deep/path/containing/a/composer.lock": &fstest.MapFile{Data: []byte("{}")},
	}

	resultChan := make(chan resultContext)
	err := analyze(context.TODO(), testFs, ".", resultChan)
	assert.NoError(t, err)

	r := <-resultChan
	assert.Equal(t, r.Analyzer.Name(), "apps")
	require.Len(t, r.results, 1)
	assert.Equal(t, ".", r.results[0].Payload)
	assert.Equal(t, "package-lock.json", r.results[0].Reason)
}
