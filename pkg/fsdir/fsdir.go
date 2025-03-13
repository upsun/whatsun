package fsdir

import (
	"io/fs"
	"unsafe"
)

// FSDir represents a single directory in a filesystem.
type FSDir struct {
	fs   fs.FS
	path string
	id   uintptr // Unique ID of the filesystem to be used for comparison (e.g. as a cache key).
}

func New(fsys fs.FS, path string) FSDir {
	return FSDir{fs: fsys, path: path, id: uintptr(unsafe.Pointer(&fsys))}
}

func (f FSDir) ID() uintptr  { return f.id }
func (f FSDir) Path() string { return f.path }
func (f FSDir) FS() fs.FS    { return f.fs }
