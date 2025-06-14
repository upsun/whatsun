package files_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/upsun/whatsun/pkg/files"
)

// customFS wraps fstest.MapFS to allow simulating permission errors
type customFS struct {
	fs fstest.MapFS

	noPermissionFiles map[string]bool // Files that should return a permission error.
}

func (c *customFS) Open(name string) (fs.File, error) {
	if c.noPermissionFiles[name] {
		return nil, fs.ErrPermission
	}
	return c.fs.Open(name)
}

func TestReadMultiple(t *testing.T) {
	file1Content := "content of file1"
	file2Content := "content of file2 which is longer"
	emptyContent := ""

	// Create an in-memory filesystem for testing.
	baseFS := fstest.MapFS{
		"file1.txt": &fstest.MapFile{
			Data: []byte(file1Content),
			Mode: 0644,
		},
		"file2.txt": &fstest.MapFile{
			Data: []byte(file2Content),
			Mode: 0644,
		},
		"empty.txt": &fstest.MapFile{
			Data: []byte(emptyContent),
			Mode: 0644,
		},
		"secret.txt": &fstest.MapFile{
			Data: []byte("This is a secret file"),
			Mode: 0644,
		},
		"config/settings.json": &fstest.MapFile{
			Data: []byte(`{"setting": "value"}`),
			Mode: 0644,
		},
		"logs/app.log": &fstest.MapFile{
			Data: []byte("log entries"),
			Mode: 0644,
		},
		".aiignore": &fstest.MapFile{
			Data: []byte("secret.txt\n*.log\nconfig/"),
			Mode: 0644,
		},
	}

	// Filesystem with a different .aiignore content
	fsWithWildcardIgnore := fstest.MapFS{
		"file1.txt": &fstest.MapFile{
			Data: []byte(file1Content),
			Mode: 0644,
		},
		"file2.txt": &fstest.MapFile{
			Data: []byte(file2Content),
			Mode: 0644,
		},
		"test1.go": &fstest.MapFile{
			Data: []byte("package main"),
			Mode: 0644,
		},
		"test2.go": &fstest.MapFile{
			Data: []byte("package main"),
			Mode: 0644,
		},
		".aiignore": &fstest.MapFile{
			Data: []byte("*.go"),
			Mode: 0644,
		},
	}

	// Filesystem with .aiexclude instead of .aiignore
	fsWithAIExclude := fstest.MapFS{
		"file1.txt": &fstest.MapFile{
			Data: []byte(file1Content),
			Mode: 0644,
		},
		"file2.txt": &fstest.MapFile{
			Data: []byte(file2Content),
			Mode: 0644,
		},
		"private.txt": &fstest.MapFile{
			Data: []byte("private content"),
			Mode: 0644,
		},
		"data/sensitive.json": &fstest.MapFile{
			Data: []byte(`{"sensitive": "data"}`),
			Mode: 0644,
		},
		".aiexclude": &fstest.MapFile{
			Data: []byte("private.txt\ndata/"),
			Mode: 0644,
		},
	}

	// Filesystem with both .aiignore and .aiexclude
	fsWithBothIgnoreFiles := fstest.MapFS{
		"file1.txt": &fstest.MapFile{
			Data: []byte(file1Content),
			Mode: 0644,
		},
		"file2.txt": &fstest.MapFile{
			Data: []byte(file2Content),
			Mode: 0644,
		},
		"doc1.md": &fstest.MapFile{
			Data: []byte("# Documentation"),
			Mode: 0644,
		},
		"doc2.md": &fstest.MapFile{
			Data: []byte("# More Documentation"),
			Mode: 0644,
		},
		"temp.txt": &fstest.MapFile{
			Data: []byte("temporary file"),
			Mode: 0644,
		},
		".aiignore": &fstest.MapFile{
			Data: []byte("*.md"),
			Mode: 0644,
		},
		".aiexclude": &fstest.MapFile{
			Data: []byte("temp.txt"),
			Mode: 0644,
		},
	}

	// Wrap baseFS in a custom filesystem that simulates permission errors.
	testFS := &customFS{
		fs: baseFS,
		noPermissionFiles: map[string]bool{
			"no-permission.txt": true,
		},
	}

	tests := []struct {
		name      string
		fsys      fs.FS
		maxLength int
		filenames []string
		want      []files.FileData
		wantErr   bool
	}{
		// Existing test cases...
		{
			name:      "read existing files",
			fsys:      testFS,
			maxLength: 100,
			filenames: []string{"file1.txt", "file2.txt"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
				{
					Name:      "file2.txt",
					Content:   file2Content,
					Size:      int64(len(file2Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "file truncation",
			fsys:      testFS,
			maxLength: 10,
			filenames: []string{"file1.txt", "file2.txt"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content[:10],
					Size:      int64(len(file1Content)),
					Truncated: true,
					Cleaned:   false,
				},
				{
					Name:      "file2.txt",
					Content:   file2Content[:10],
					Size:      int64(len(file2Content)),
					Truncated: true,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "nonexistent file",
			fsys:      testFS,
			maxLength: 100,
			filenames: []string{"file1.txt", "nonexistent.txt", "file2.txt"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
				{
					Name:      "file2.txt",
					Content:   file2Content,
					Size:      int64(len(file2Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "permission error",
			fsys:      testFS,
			maxLength: 100,
			filenames: []string{"file1.txt", "no-permission.txt", "file2.txt"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
				{
					Name:      "file2.txt",
					Content:   file2Content,
					Size:      int64(len(file2Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "empty file",
			fsys:      testFS,
			maxLength: 100,
			filenames: []string{"file*.txt", "empty.txt", "file2.txt"},
			want: []files.FileData{
				{
					Name:      "empty.txt",
					Content:   emptyContent,
					Size:      int64(len(emptyContent)),
					Truncated: false,
					Cleaned:   false,
				},
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
				{
					Name:      "file2.txt",
					Content:   file2Content,
					Size:      int64(len(file2Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "no files",
			fsys:      testFS,
			maxLength: 100,
			filenames: []string{},
			want:      []files.FileData{},
			wantErr:   false,
		},

		// New test cases for .aiignore functionality
		{
			name:      "ignore exact file match",
			fsys:      testFS,
			maxLength: 100,
			filenames: []string{"file1.txt", "secret.txt", "file2.txt"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
				{
					Name:      "file2.txt",
					Content:   file2Content,
					Size:      int64(len(file2Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "ignore wildcard extension",
			fsys:      testFS,
			maxLength: 100,
			filenames: []string{"file1.txt", "logs/app.log"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "ignore directory",
			fsys:      testFS,
			maxLength: 100,
			filenames: []string{"file1.txt", "config/settings.json"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "ignore all specified files",
			fsys:      testFS,
			maxLength: 100,
			filenames: []string{"secret.txt", "logs/app.log", "config/settings.json"},
			want:      []files.FileData{},
			wantErr:   false,
		},
		{
			name:      "wildcard file extension ignore",
			fsys:      fsWithWildcardIgnore,
			maxLength: 100,
			filenames: []string{"file1.txt", "test1.go", "test2.go", "file2.txt"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
				{
					Name:      "file2.txt",
					Content:   file2Content,
					Size:      int64(len(file2Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},

		// New test cases for .aiexclude functionality
		{
			name:      "exclude file with .aiexclude",
			fsys:      fsWithAIExclude,
			maxLength: 100,
			filenames: []string{"file1.txt", "private.txt", "file2.txt"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
				{
					Name:      "file2.txt",
					Content:   file2Content,
					Size:      int64(len(file2Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "exclude directory with .aiexclude",
			fsys:      fsWithAIExclude,
			maxLength: 100,
			filenames: []string{"file1.txt", "data/sensitive.json"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "both aiignore and aiexclude active",
			fsys:      fsWithBothIgnoreFiles,
			maxLength: 100,
			filenames: []string{"file1.txt", "doc1.md", "temp.txt", "file2.txt", "doc2.md"},
			want: []files.FileData{
				{
					Name:      "file1.txt",
					Content:   file1Content,
					Size:      int64(len(file1Content)),
					Truncated: false,
					Cleaned:   false,
				},
				{
					Name:      "file2.txt",
					Content:   file2Content,
					Size:      int64(len(file2Content)),
					Truncated: false,
					Cleaned:   false,
				},
			},
			wantErr: false,
		},
		{
			name:      "only excluded files",
			fsys:      fsWithBothIgnoreFiles,
			maxLength: 100,
			filenames: []string{"doc1.md", "temp.txt", "doc2.md"},
			want:      []files.FileData{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := files.ReadMultiple(tt.fsys, tt.maxLength, tt.filenames...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
