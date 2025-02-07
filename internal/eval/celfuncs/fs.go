package celfuncs

import (
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
)

// AllFileFunctions returns CEL functions for reading an fs.FS filesystem.
func AllFileFunctions(fsys *fs.FS, root *string) []cel.EnvOption {
	if root == nil {
		dot := "."
		root = &dot
	}
	return []cel.EnvOption{
		FileContains(fsys, root),
		FileExists(fsys, root),
		FileGlob(fsys, root),
		FileExistsRegex(fsys, root),
		FileIsDir(fsys, root),
		FileRead(fsys, root),
	}
}

// FileContains defines a CEL function `file.contains(path, substr) -> bool`.
func FileContains(fsys *fs.FS, root *string) cel.EnvOption {
	FuncComments["file.contains"] = "Check whether a file contains a substring"

	return stringStringReturnsBoolErr("file.contains", func(path, substr string) (bool, error) {
		b, err := fs.ReadFile(*fsys, filepath.Join(*root, path))
		if err != nil {
			return false, err
		}
		return strings.Contains(string(b), substr), nil
	})
}

// FileRead defines a CEL function `file.read(path) -> bytes`.
func FileRead(fsys *fs.FS, root *string) cel.EnvOption {
	FuncComments["file.read"] = "Read a file"

	return stringReturnsBytesErr("file.read", func(path string) ([]byte, error) {
		return fs.ReadFile(*fsys, filepath.Join(*root, path))
	})
}

// FileExists defines a CEL function `file.exists(path) -> bool`.
func FileExists(fsys *fs.FS, root *string) cel.EnvOption {
	FuncComments["file.exists"] = "Check whether a file exists"

	return stringReturnsBoolErr("file.exists", func(path string) (bool, error) {
		_, err := fs.Stat(*fsys, filepath.Join(*root, path))
		if err != nil {
			return false, ignoreNotExists(err)
		}
		return true, nil
	})
}

// FileGlob defines a CEL function `file.glob(pattern) -> list`.
func FileGlob(fsys *fs.FS, root *string) cel.EnvOption {
	FuncComments["file.glob"] = "List files matching a glob pattern"

	return stringReturnsListErr("file.glob", func(pattern string) ([]string, error) {
		return fs.Glob(*fsys, filepath.Join(*root, pattern))
	})
}

// FileExistsRegex defines a CEL function `file.existsRegex(path) -> bool`.
func FileExistsRegex(fsys *fs.FS, root *string) cel.EnvOption {
	FuncComments["file.existsRegex"] = "Check if files exist matching a regular expression"

	return stringReturnsBoolErr("file.existsRegex", func(pattern string) (bool, error) {
		entries, err := fs.ReadDir(*fsys, *root)
		if err != nil {
			return false, err
		}
		rx, err := regexp.Compile(pattern)
		if err != nil {
			return false, err
		}
		for _, e := range entries {
			if rx.MatchString(e.Name()) {
				return true, nil
			}
		}
		return false, nil
	})
}

// FileIsDir defines a CEL function `file.isDir(path) -> bool`.
func FileIsDir(fsys *fs.FS, root *string) cel.EnvOption {
	FuncComments["file.isDir"] = "Check if a file exists and is a directory"

	return stringReturnsBoolErr("file.isDir", func(path string) (bool, error) {
		stat, err := fs.Stat(*fsys, filepath.Join(*root, path))
		if err != nil {
			return false, ignoreNotExists(err)
		}
		return stat.IsDir(), nil
	})
}

func ignoreNotExists(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
