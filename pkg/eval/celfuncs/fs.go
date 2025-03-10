package celfuncs

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/google/cel-go/cel"
)

const (
	fsVariable   = "fs"
	pathVariable = "path"
)

// FilesystemInput returns a CEL program input for a filesystem named "fs", and a file path variable named "path".
// This can only be used alongside the FilesystemVariables options.
func FilesystemInput(fsys fs.FS, root string) map[string]any {
	return map[string]any{
		pathVariable: root,
		fsVariable:   filesystemWrapper{FS: fsys, Path: root, ID: uintptr(unsafe.Pointer(&fsys))},
	}
}

type filesystemWrapper struct {
	FS   fs.FS
	Path string
	ID   uintptr
}

// FilesystemVariables returns CEL options to create variables corresponding to FilesystemInput.
func FilesystemVariables() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Variable(pathVariable, cel.StringType),
		cel.Variable(fsVariable, cel.DynType),
	}
}

// AllFileOptions returns CEL functions for reading an fs.FS filesystem.
// This can only be used alongside the FilesystemVariables options.
func AllFileOptions(docs *Docs) []cel.EnvOption {
	return []cel.EnvOption{
		FileContains(docs),
		FileExists(docs),
		FileGlob(docs),
		FileIsDir(docs),
		FileRead(docs),
	}
}

// fsUnaryFunction returns a CEL environment option for a receiver method on the filesystem variable ("fs") that takes 1 argument.
func fsUnaryFunction[ARG any, RET any](name string, argType *cel.Type, returnType *cel.Type, f func(filesystemWrapper, ARG) (RET, error)) cel.EnvOption {
	return unaryReceiverFunction[filesystemWrapper, ARG, RET](fsVariable, name, []*cel.Type{cel.DynType, argType}, returnType, f)
}

// fsBinaryFunction returns a CEL environment option for a receiver method on the filesystem variable ("fs") that takes 2 arguments.
func fsBinaryFunction[A1 any, A2 any, RET any](name string, argTypes []*cel.Type, returnType *cel.Type, f func(filesystemWrapper, A1, A2) (RET, error)) cel.EnvOption {
	return binaryReceiverFunction[filesystemWrapper, A1, A2, RET](fsVariable, name, append([]*cel.Type{cel.DynType}, argTypes...), returnType, f)
}

func FileExists(docs *Docs) cel.EnvOption {
	docs.AddFunction("fileExists", FuncDoc{
		Comment: "Check whether a file exists",
		Args:    []ArgDoc{{"fs", ""}, {"filename", ""}},
	})

	return fsUnaryFunction[string, bool]("fileExists", cel.StringType, cel.BoolType, func(fsWrapper filesystemWrapper, name string) (bool, error) {
		_, err := fs.Stat(fsWrapper.FS, filepath.Join(fsWrapper.Path, name))
		if err != nil {
			return false, ignoreNotExists(err)
		}
		return true, nil
	})
}

func FileContains(docs *Docs) cel.EnvOption {
	docs.AddFunction("fileContains", FuncDoc{
		Comment: "Check whether a file contains a substring",
		Args: []ArgDoc{
			{"fs", ""},
			{"filename", ""},
			{"substr", ""},
		},
	})

	return fsBinaryFunction[string, string, bool](
		"fileContains",
		[]*cel.Type{cel.StringType, cel.StringType},
		cel.BoolType,
		func(fsWrapper filesystemWrapper, filename, substr string) (bool, error) {
			b, err := fs.ReadFile(fsWrapper.FS, filepath.Join(fsWrapper.Path, filename))
			if err != nil {
				return false, err
			}
			return strings.Contains(string(b), substr), nil
		},
	)
}

func FileGlob(docs *Docs) cel.EnvOption {
	docs.AddFunction("glob", FuncDoc{
		Comment: "List files matching a glob pattern",
		Args:    []ArgDoc{{"fs", ""}, {"pattern", ""}},
	})

	return fsUnaryFunction[string, []string]("glob", cel.StringType, cel.ListType(cel.StringType), func(fsWrapper filesystemWrapper, pattern string) ([]string, error) {
		return fs.Glob(fsWrapper.FS, filepath.Join(fsWrapper.Path, pattern))
	})
}

func FileRead(docs *Docs) cel.EnvOption {
	docs.AddFunction("read", FuncDoc{
		Comment: "Read a file",
		Args:    []ArgDoc{{"fs", ""}, {"filename", ""}},
	})

	return fsUnaryFunction[string, []byte]("read", cel.StringType, cel.BytesType, func(fsWrapper filesystemWrapper, path string) ([]byte, error) {
		return fs.ReadFile(fsWrapper.FS, filepath.Join(fsWrapper.Path, path))
	})
}

func FileIsDir(docs *Docs) cel.EnvOption {
	docs.AddFunction("isDir", FuncDoc{
		Comment: "Check if a file exists and is a directory",
		Args:    []ArgDoc{{"fs", ""}, {"name", ""}},
	})

	return fsUnaryFunction[string, bool]("isDir", cel.StringType, cel.BoolType, func(fsWrapper filesystemWrapper, name string) (bool, error) {
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
