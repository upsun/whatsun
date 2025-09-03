package searchfs_test

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upsun/whatsun/pkg/searchfs"
)

var testFs = fstest.MapFS{
	"1/a":         &fstest.MapFile{},
	"1/2/a":       &fstest.MapFile{},
	"1/2/3/a":     &fstest.MapFile{},
	"1/2/3/4/a":   &fstest.MapFile{},
	"1/2/3/4/5/a": &fstest.MapFile{},

	"1/b":         &fstest.MapFile{},
	"1/2/b":       &fstest.MapFile{},
	"1/2/3/b":     &fstest.MapFile{},
	"1/2/3/4/b":   &fstest.MapFile{},
	"1/2/3/4/5/b": &fstest.MapFile{},

	"1/abc":         &fstest.MapFile{},
	"1/2/abc":       &fstest.MapFile{},
	"1/2/3/abc":     &fstest.MapFile{},
	"1/2/3/4/abc":   &fstest.MapFile{},
	"1/2/3/4/5/abc": &fstest.MapFile{},
}

func TestFS(t *testing.T) {
	type caseType struct {
		path      string
		openErr   error
		statErr   error
		globErr   error
		globCount int
	}

	var cases = make([]caseType, 0, 25)
	for _, d := range genDirs(1, 5) {
		cases = append(cases,
			caseType{path: d, globCount: 1},
			caseType{path: d + "/a", globCount: 1},
			caseType{path: d + "/b", globCount: 1},
			caseType{path: d + "/a*", openErr: fs.ErrNotExist, statErr: fs.ErrNotExist, globCount: 2},
			caseType{path: d + "/nonexistent", openErr: fs.ErrNotExist, statErr: fs.ErrNotExist},
		)
	}

	fsys := searchfs.New(testFs)
	for _, c := range cases {
		_, err := fsys.Open(c.path)
		if c.openErr == nil {
			assert.NoError(t, err, "open %s (should succeed)", c.path)
		} else {
			assert.ErrorIs(t, err, c.openErr, "open %s (should fail)", c.path)
		}

		_, err = fs.Stat(fsys, c.path)
		if c.statErr == nil {
			assert.NoError(t, err, "stat %s (should succeed)", c.path)
		} else {
			assert.ErrorIs(t, err, c.statErr, "stat %s (should fail)", c.path)
		}

		l, err := fs.Glob(fsys, c.path)
		if c.globErr == nil {
			require.NoError(t, err, "glob %s (should succeed)", c.path)
			assert.Len(t, l, c.globCount, "glob %s", c.path)
		} else {
			assert.ErrorIs(t, err, c.globErr, "glob %s (should fail)", c.path)
		}
	}
}

func genDirs(from, to int) []string {
	if from > to {
		return nil
	}
	var paths = make([]string, to-from+1)
	var builder strings.Builder
	i := 0
	for n := from; n <= to; n++ {
		if n > from {
			builder.WriteString("/")
		}
		builder.WriteString(fmt.Sprintf("%d", n))
		paths[i] = builder.String()
		i++
	}
	return paths
}
