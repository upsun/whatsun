package celfuncs

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/google/cel-go/cel"
)

type filesystemWrapper struct {
	FS   fs.FS
	Path string
	ID   uintptr
}

const fsVariable = "fs"

func FilesystemVariable() cel.EnvOption {
	return cel.Variable(fsVariable, cel.DynType)
}

func FilesystemInput(fsys fs.FS, root string) map[string]any {
	return map[string]any{
		fsVariable: filesystemWrapper{FS: fsys, Path: root, ID: uintptr(unsafe.Pointer(&fsys))},
	}
}

// AllFileFunctions returns CEL functions for reading an fs.FS filesystem.
// This can only be used alongside the FilesystemVariable option.
func AllFileFunctions() []cel.EnvOption {
	return []cel.EnvOption{
		FileContains(),
		FileExists(),
		FileGlob(),
		FileIsDir(),
		FilePath(),
		FileRead(),
	}
}

func FilePath() cel.EnvOption {
	FuncDocs["path"] = FuncDoc{
		Comment: "Get the current file path",
		Args:    []ArgDoc{{"fs", ""}},
	}

	return fsReturnsStringErr("path", func(fsWrapper filesystemWrapper) (string, error) {
		return fsWrapper.Path, nil
	})
}

func FileExists() cel.EnvOption {
	FuncDocs["fileExists"] = FuncDoc{
		Comment: "Check whether a file exists",
		Args:    []ArgDoc{{"fs", ""}, {"filename", ""}},
	}

	return fsStringReturnsBoolErr("fileExists", func(fsWrapper filesystemWrapper, filename string) (bool, error) {
		_, err := fs.Stat(fsWrapper.FS, filepath.Join(fsWrapper.Path, filename))
		if err != nil {
			return false, ignoreNotExists(err)
		}
		return true, nil
	})
}

func FileContains() cel.EnvOption {
	FuncDocs["fileContains"] = FuncDoc{
		Comment: "Check whether a file contains a substring",
		Args: []ArgDoc{
			{"fs", ""},
			{"filename", ""},
			{"substr", ""},
		},
	}

	return fsStringStringReturnsBoolErr("fileContains", func(fsWrapper filesystemWrapper, filename, substr string) (bool, error) {
		b, err := fs.ReadFile(fsWrapper.FS, filepath.Join(fsWrapper.Path, filename))
		if err != nil {
			return false, err
		}
		return strings.Contains(string(b), substr), nil
	})
}

func FileGlob() cel.EnvOption {
	FuncDocs["glob"] = FuncDoc{
		Comment: "List files matching a glob pattern",
		Args:    []ArgDoc{{"fs", ""}, {"pattern", ""}},
	}

	return fsStringReturnsListErr("glob", func(fsWrapper filesystemWrapper, pattern string) ([]string, error) {
		return fs.Glob(fsWrapper.FS, filepath.Join(fsWrapper.Path, pattern))
	})
}

func FileRead() cel.EnvOption {
	FuncDocs["read"] = FuncDoc{
		Comment: "Read a file",
		Args:    []ArgDoc{{"fs", ""}, {"filename", ""}},
	}

	return fsStringReturnsBytesErr("read", func(fsWrapper filesystemWrapper, path string) ([]byte, error) {
		return fs.ReadFile(fsWrapper.FS, filepath.Join(fsWrapper.Path, path))
	})
}

func FileIsDir() cel.EnvOption {
	FuncDocs["isDir"] = FuncDoc{
		Comment: "Check if a file exists and is a directory",
		Args:    []ArgDoc{{"fs", ""}, {"name", ""}},
	}

	return fsStringReturnsBoolErr("isDir", func(fsWrapper filesystemWrapper, name string) (bool, error) {
		stat, err := fs.Stat(fsWrapper.FS, filepath.Join(fsWrapper.Path, name))
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
