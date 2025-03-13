package dep_test

import (
	"io/fs"
	"unsafe"
)

type filesystemWrapper struct {
	fs   fs.FS
	path string
	id   uintptr
}

func newFilesystemWrapper(fsys fs.FS, path string) filesystemWrapper {
	return filesystemWrapper{fs: fsys, path: path, id: uintptr(unsafe.Pointer(&fsys))}
}

func (f filesystemWrapper) ID() uintptr  { return f.id }
func (f filesystemWrapper) Path() string { return f.path }
func (f filesystemWrapper) FS() fs.FS    { return f.fs }
