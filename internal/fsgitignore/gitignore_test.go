package fsgitignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGlobalIgnorePatterns(t *testing.T) {
	tests := []struct {
		name             string
		setupFunc        func(t *testing.T) (cleanup func())
		expectedError    bool
		expectedPatterns bool
	}{
		{
			name: "no global gitignore file",
			setupFunc: func(t *testing.T) func() {
				// Create a temporary home directory that doesn't have .gitignore
				tempDir := t.TempDir()
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				return func() {
					os.Setenv("HOME", oldHome)
				}
			},
			expectedError:    false,
			expectedPatterns: false,
		},
		{
			name: "global gitignore file exists",
			setupFunc: func(t *testing.T) func() {
				// Create a temporary home directory with .gitignore
				tempDir := t.TempDir()
				gitignoreContent := `# Global gitignore
*.log
.DS_Store
`
				require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignoreContent), 0600))

				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				return func() {
					os.Setenv("HOME", oldHome)
				}
			},
			expectedError:    false,
			expectedPatterns: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupFunc(t)
			defer cleanup()

			patterns, err := GetGlobalIgnorePatterns()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedPatterns {
				assert.NotEmpty(t, patterns)
				// Check that at least one pattern was parsed
				assert.True(t, len(patterns) > 0)
			} else {
				assert.Empty(t, patterns)
			}
		})
	}
}

func TestGetGlobalGitignorePath(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (cleanup func())
		expectPath  string
		expectEmpty bool
	}{
		{
			name: "no gitignore file found",
			setupFunc: func(t *testing.T) func() {
				tempDir := t.TempDir()
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				return func() {
					os.Setenv("HOME", oldHome)
				}
			},
			expectEmpty: true,
		},
		{
			name: "default gitignore file exists",
			setupFunc: func(t *testing.T) func() {
				tempDir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("*.log"), 0600))
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				return func() {
					os.Setenv("HOME", oldHome)
				}
			},
			expectPath: "/.gitignore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupFunc(t)
			defer cleanup()

			path, err := getGlobalGitignorePath()
			assert.NoError(t, err)

			if tt.expectEmpty {
				assert.Empty(t, path)
			} else if tt.expectPath != "" {
				assert.True(t, strings.HasSuffix(path, tt.expectPath))
			}
		})
	}
}

func TestParseIgnoreFile(t *testing.T) {
	content := `# This is a comment
*.log
# Another comment

.DS_Store
build/
`
	reader := strings.NewReader(content)
	patterns := ParseIgnoreFile(reader, nil)

	assert.Len(t, patterns, 3) // Should have 3 patterns (comments and empty lines ignored)

	// Test that patterns work correctly by testing matches
	assert.NotEqual(t, gitignore.NoMatch, patterns[0].Match([]string{"test.log"}, false))
	assert.NotEqual(t, gitignore.NoMatch, patterns[1].Match([]string{".DS_Store"}, false))
	assert.NotEqual(t, gitignore.NoMatch, patterns[2].Match([]string{"build"}, true))
}
