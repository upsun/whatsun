// Package fsgitignore adapts go-git's functions for use with an io/fs filesystem.
package fsgitignore

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

const (
	commentPrefix   = "#"
	gitDir          = ".git"
	gitignoreFile   = ".gitignore"
	infoExcludeFile = gitDir + "/info/exclude"
)

// Split splits a file path into segments, i.e. the format the upstream gitignore package prefers.
func Split(path string) []string {
	if path == "." {
		return []string{}
	}
	return strings.Split(filepath.Clean(path), string(os.PathSeparator))
}

// ParsePatterns turns string patterns into parsed versions.
// The domain is usually a relative path split into segments (see Split).
func ParsePatterns(patterns, domain []string) []gitignore.Pattern {
	var parsed = make([]gitignore.Pattern, len(patterns))
	for i, pattern := range patterns {
		parsed[i] = gitignore.ParsePattern(pattern, domain)
	}
	return parsed
}

// ParseIgnoreFiles parses the gitignore files in a single directory.
func ParseIgnoreFiles(fsys fs.FS, path string) ([]gitignore.Pattern, error) {
	var ps []gitignore.Pattern
	domain := Split(path)
	for _, filename := range []string{gitignoreFile, infoExcludeFile} {
		if err := handleIfExists(fsys, path, filename, func(r io.Reader) {
			ps = append(ps, ParseIgnoreFile(r, domain)...)
		}); err != nil {
			return nil, err
		}
	}
	return ps, nil
}

// ParseIgnoreFile reads gitignore patterns from a specific file.
// The domain is usually a relative path split into segments (see Split).
// See ParsePatterns to handle a string slice directly.
func ParseIgnoreFile(r io.Reader, domain []string) []gitignore.Pattern {
	var ps []gitignore.Pattern
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s := scanner.Text()
		if !strings.HasPrefix(s, commentPrefix) && len(strings.TrimSpace(s)) > 0 {
			ps = append(ps, gitignore.ParsePattern(s, domain))
		}
	}
	return ps
}

// handleIfExists opens a file, ignoring errors if the file does not exist or cannot be accessed, and runs a function.
func handleIfExists(fsys fs.FS, path, filename string, handler func(r io.Reader)) error {
	f, err := fsys.Open(filepath.Join(path, filename))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if errors.Is(err, syscall.ENOTDIR) {
			return nil
		}
		return err
	}
	defer f.Close()

	handler(f)
	return nil
}

// getGlobalGitignorePath returns the path to the global gitignore file.
// It first checks for ~/.gitignore, then falls back to git config core.excludesFile.
func getGlobalGitignorePath() (string, error) {
	// Get home directory first
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Check for default ~/.gitignore first
	defaultPath := filepath.Join(home, ".gitignore")
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, nil
	}

	// Fall back to checking git config
	cmd := exec.Command("git", "config", "--global", "--get", "core.excludesFile")
	output, err := cmd.Output()
	if err == nil {
		path := strings.TrimSpace(string(output))
		if path != "" {
			// Expand ~ to home directory if needed
			if strings.HasPrefix(path, "~/") {
				path = filepath.Join(home, path[2:])
			}
			return path, nil
		}
	}

	// No global gitignore file found
	return "", nil
}

// GetGlobalIgnorePatterns parses and returns global gitignore patterns from the user's global gitignore file.
func GetGlobalIgnorePatterns() ([]gitignore.Pattern, error) {
	globalPath, err := getGlobalGitignorePath()
	if err != nil {
		return nil, err
	}

	if globalPath == "" {
		return nil, nil // No global gitignore file found
	}

	file, err := os.Open(globalPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil // No global gitignore file is fine
		}
		return nil, err
	}
	defer file.Close()

	return ParseIgnoreFile(file, nil), nil
}

//go:embed gitignore-defaults
var defaultIgnoreFile []byte

var defaultIgnores []gitignore.Pattern
var parsedDefaultIgnores sync.Once

// GetDefaultIgnorePatterns parses and returns common ignore patterns.
func GetDefaultIgnorePatterns() []gitignore.Pattern {
	parsedDefaultIgnores.Do(func() {
		defaultIgnores = ParseIgnoreFile(bytes.NewReader(defaultIgnoreFile), nil)
	})
	return defaultIgnores
}
