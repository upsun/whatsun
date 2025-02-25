// Package fsgitignore adapts go-git's functions for use with an io/fs filesystem.
package fsgitignore

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
func ParsePatterns(patterns, path []string) []gitignore.Pattern {
	var parsed = make([]gitignore.Pattern, len(patterns))
	for i, pattern := range patterns {
		parsed[i] = gitignore.ParsePattern(pattern, path)
	}
	return parsed
}

// ParseIgnoreFiles parses the gitignore files in a single directory.
func ParseIgnoreFiles(fsys fs.FS, path string) ([]gitignore.Pattern, error) {
	var ps []gitignore.Pattern
	excludes, err := parseIgnoreFile(fsys, path, infoExcludeFile)
	if err != nil {
		return nil, err
	}
	ps = append(ps, excludes...)
	ignores, err := parseIgnoreFile(fsys, path, gitignoreFile)
	if err != nil {
		return nil, err
	}
	return append(ps, ignores...), nil
}

// parseIgnoreFile reads a specific git ignore file.
func parseIgnoreFile(fsys fs.FS, path, ignoreFile string) (ps []gitignore.Pattern, err error) {
	f, err := fsys.Open(filepath.Join(path, ignoreFile))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		if errors.Is(err, syscall.ENOTDIR) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		if !strings.HasPrefix(s, commentPrefix) && len(strings.TrimSpace(s)) > 0 {
			ps = append(ps, gitignore.ParsePattern(s, Split(path)))
		}
	}

	return
}
