package digest

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTree(t *testing.T) {
	fsys := fstest.MapFS{
		".gitignore":            &fstest.MapFile{Data: []byte("custom-ignore")},
		"a.txt":                 &fstest.MapFile{},
		"b.txt":                 &fstest.MapFile{},
		"dir/c.txt":             &fstest.MapFile{},
		"custom-ignore/foo.txt": &fstest.MapFile{},
		"node_modules/foo.txt":  &fstest.MapFile{},
		"vendor/foo/bar.txt":    &fstest.MapFile{},
	}

	cases := []struct {
		name   string
		config TreeConfig
		expect []string
	}{
		{
			"default",
			TreeConfig{},
			[]string{
				".",
				"├ .gitignore",
				"├ a.txt",
				"├ b.txt",
				"└ dir",
				"  └ c.txt",
			},
		},
		{
			"plain",
			TreeConfig{
				EntryConnector:        "|-",
				LastEntryConnector:    "|_",
				ContinuationConnector: "|",
			},
			[]string{
				".",
				"|- .gitignore",
				"|- a.txt",
				"|- b.txt",
				"|_ dir",
				"  |_ c.txt",
			},
		},
		{
			"whitespace",
			TreeConfig{
				EntryConnector:        " ",
				LastEntryConnector:    " ",
				ContinuationConnector: " ",
				DirectorySuffix:       "/",
			},
			[]string{
				"./",
				"  .gitignore",
				"  a.txt",
				"  b.txt",
				"  dir/",
				"    c.txt",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := GetTree(fsys, c.config)
			require.NoError(t, err)
			assert.EqualValues(t, c.expect, result)
		})
	}
}

func TestGetTree_MaxEntriesPerLevel(t *testing.T) {
	fsys := fstest.MapFS{
		"fileA.txt":           &fstest.MapFile{},
		"fileB.txt":           &fstest.MapFile{},
		"fileC.txt":           &fstest.MapFile{},
		"fileD.txt":           &fstest.MapFile{},
		"dir1/fileA.txt":      &fstest.MapFile{},
		"dir1/fileB.txt":      &fstest.MapFile{},
		"dir1/fileC.txt":      &fstest.MapFile{},
		"dir1/fileD.txt":      &fstest.MapFile{},
		"dir1/dir2/fileA.txt": &fstest.MapFile{},
		"dir1/dir2/fileB.txt": &fstest.MapFile{},
		"dir1/dir2/fileC.txt": &fstest.MapFile{},
		"dir1/dir2/fileD.txt": &fstest.MapFile{},
	}

	got, err := GetTree(fsys, TreeConfig{
		MaxEntries:         8,
		MaxEntriesPerLevel: 0.5,
	})
	require.NoError(t, err)

	assert.EqualValues(t, []string{
		".",
		"├ dir1",
		"│ ├ dir2",
		"│ │ ├ fileA.txt",
		"│ │ ├ fileB.txt",
		"│ │ └ ... (2 more)",
		"│ ├ fileA.txt",
		"│ ├ fileB.txt",
		"│ ├ fileC.txt",
		"│ └ fileD.txt",
		"├ fileA.txt",
		"├ fileB.txt",
		"├ fileC.txt",
		"└ fileD.txt",
	}, got)
}
