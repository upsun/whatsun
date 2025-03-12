// Package searchfs provides a filesystem implementation that calls ReadDir on every directory and caches the result.
// It aims to avoid unnecessary "stat" calls when looking for (or opening) many files in a tree.
// The filenames provided must be already cleaned (via filepath.Clean).
package searchfs

import (
	"io/fs"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type FS struct {
	baseFS     fs.FS
	rootInfo   atomic.Value // fs.FileInfo of the root directory
	dirEntries sync.Map
}

func New(base fs.FS) *FS {
	return &FS{baseFS: base}
}

func (sfs *FS) getDirEntries(name string) []fs.DirEntry {
	if de, ok := sfs.dirEntries.Load(name); ok {
		return de.([]fs.DirEntry) //nolint:errcheck // the type is known
	}
	return nil
}

func (sfs *FS) setDirEntries(name string, entries []fs.DirEntry) {
	sfs.dirEntries.Store(name, entries)
}

func (sfs *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries := sfs.getDirEntries(name)
	if entries == nil {
		e, err := fs.ReadDir(sfs.baseFS, name)
		if err != nil {
			return nil, err
		}
		sfs.setDirEntries(name, e)
		entries = e
	}
	return entries, nil
}

func (sfs *FS) Stat(name string) (fs.FileInfo, error) {
	if name == "." {
		if fi := sfs.rootInfo.Load(); fi != nil {
			return fi.(fs.FileInfo), nil //nolint:errcheck // the type is known
		}
		fi, err := fs.Stat(sfs.baseFS, ".")
		if err != nil {
			return nil, err
		}
		sfs.rootInfo.Store(fi)
		return fi, nil
	}
	entries, err := sfs.ReadDir(filepath.Dir(name))
	if err != nil {
		return nil, err
	}
	basename := filepath.Base(name)
	for _, e := range entries {
		if e.Name() == basename {
			return e.Info()
		}
	}
	return nil, fs.ErrNotExist
}

func (sfs *FS) Open(name string) (fs.File, error) {
	if _, err := sfs.Stat(name); err != nil {
		return nil, err
	}
	return sfs.baseFS.Open(name)
}
