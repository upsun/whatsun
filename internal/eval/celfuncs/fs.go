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

func FileContains(fsys *fs.FS, root *string) cel.EnvOption {
	FuncDocs["file.contains"] = FuncDoc{
		Comment: "Check whether a file contains a substring",
		Args: []ArgDoc{
			{"filename", ""},
			{"substr", ""},
		},
	}

	return stringStringReturnsBoolErr("file.contains", func(filename, substr string) (bool, error) {
		b, err := fs.ReadFile(*fsys, filepath.Join(*root, filename))
		if err != nil {
			return false, err
		}
		return strings.Contains(string(b), substr), nil
	})
}

func FileRead(fsys *fs.FS, root *string) cel.EnvOption {
	FuncDocs["file.read"] = FuncDoc{
		Comment: "Read a file",
		Args:    []ArgDoc{{"filename", ""}},
	}

	return stringReturnsBytesErr("file.read", func(path string) ([]byte, error) {
		return fs.ReadFile(*fsys, filepath.Join(*root, path))
	})
}

func FileExists(fsys *fs.FS, root *string) cel.EnvOption {
	FuncDocs["file.exists"] = FuncDoc{
		Comment: "Check whether a file exists",
		Args:    []ArgDoc{{"filename", ""}},
	}

	return stringReturnsBoolErr("file.exists", func(filename string) (bool, error) {
		_, err := fs.Stat(*fsys, filepath.Join(*root, filename))
		if err != nil {
			return false, ignoreNotExists(err)
		}
		return true, nil
	})
}

func FileGlob(fsys *fs.FS, root *string) cel.EnvOption {
	FuncDocs["file.glob"] = FuncDoc{
		Comment: "List files matching a glob pattern",
		Args:    []ArgDoc{{"pattern", ""}},
	}

	return stringReturnsListErr("file.glob", func(pattern string) ([]string, error) {
		return fs.Glob(*fsys, filepath.Join(*root, pattern))
	})
}

func FileExistsRegex(fsys *fs.FS, root *string) cel.EnvOption {
	FuncDocs["file.existsRegex"] = FuncDoc{
		Comment:     "Check if files exist matching a regular expression pattern",
		Description: "The `pattern` is a Go regular expression ([syntax overview](https://pkg.go.dev/regexp/syntax#hdr-Syntax)).",
		Args:        []ArgDoc{{"pattern", ""}},
	}

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
	FuncDocs["file.isDir"] = FuncDoc{
		Comment: "Check if a file exists and is a directory",
		Args:    []ArgDoc{{"name", ""}},
	}

	return stringReturnsBoolErr("file.isDir", func(name string) (bool, error) {
		stat, err := fs.Stat(*fsys, filepath.Join(*root, name))
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
