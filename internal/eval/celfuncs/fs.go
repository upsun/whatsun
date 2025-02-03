package celfuncs

import (
	"errors"
	"io/fs"
	"path/filepath"
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
		FileIsDir(fsys, root),
		FileRead(fsys, root),
	}
}

// FileContains defines a CEL function `file.contains(path, substr) -> bool`.
func FileContains(fsys *fs.FS, root *string) cel.EnvOption {
	return stringStringReturnsBoolErr("file.contains", func(path, substr string) (bool, error) {
		b, err := fs.ReadFile(*fsys, filepath.Join(*root, filepath.Join(*root, path)))
		if err != nil {
			return false, err
		}
		return strings.Contains(string(b), substr), nil
	})
}

// FileRead defines a CEL function `file.read(path) -> bytes`.
func FileRead(fsys *fs.FS, root *string) cel.EnvOption {
	return stringReturnsBytesErr("file.read", func(path string) ([]byte, error) {
		return fs.ReadFile(*fsys, filepath.Join(*root, filepath.Join(*root, path)))
	})
}

// FileExists defines a CEL function `file.exists(path) -> bool`.
func FileExists(fsys *fs.FS, root *string) cel.EnvOption {
	return stringReturnsBoolErr("file.exists", func(path string) (bool, error) {
		_, err := fs.Stat(*fsys, filepath.Join(*root, path))
		if err != nil {
			return false, ignoreNotExists(err)
		}
		return true, nil
	})
}

// FileIsDir defines a CEL function `file.isDir(path) -> bool`.
func FileIsDir(fsys *fs.FS, root *string) cel.EnvOption {
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
