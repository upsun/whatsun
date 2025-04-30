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
