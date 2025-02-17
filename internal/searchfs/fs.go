// Package searchfs provides a filesystem implementation that calls ReadDir on every directory and caches the result.
// It aims to avoid unnecessary "stat" calls when looking for (or opening) many files in a tree.
// The filenames provided must be already cleaned (via filepath.Clean).
package searchfs

import (
	"io/fs"
	"path/filepath"
	"sync"
)

type FS struct {
	baseFS fs.FS
	store  *store
}

type store struct {
	info       fs.FileInfo
	dirEntries map[string][]fs.DirEntry
	mux        sync.RWMutex
}

func (c *store) getInfo() fs.FileInfo {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.info
}

func (c *store) setInfo(info fs.FileInfo) {
	c.mux.Lock()
	c.info = info
	c.mux.Unlock()
}

func (c *store) get(name string) []fs.DirEntry {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if c.dirEntries == nil {
		return nil
	}
	return c.dirEntries[name]
}

func (c *store) set(name string, entries []fs.DirEntry) {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.dirEntries == nil {
		c.dirEntries = make(map[string][]fs.DirEntry)
	}
	c.dirEntries[name] = entries
}

func New(base fs.FS) FS {
	return FS{
		baseFS: base,
		store:  &store{},
	}
}

func (sfs FS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries := sfs.store.get(name)
	if entries == nil {
		e, err := fs.ReadDir(sfs.baseFS, name)
		if err != nil {
			return nil, err
		}
		sfs.store.set(name, e)
		entries = e
	}
	return entries, nil
}

func (sfs FS) Stat(name string) (fs.FileInfo, error) {
	if name == "." {
		if fi := sfs.store.getInfo(); fi != nil {
			return fi, nil
		}
		fi, err := fs.Stat(sfs.baseFS, ".")
		if err != nil {
			return nil, err
		}
		sfs.store.setInfo(fi)
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

func (sfs FS) Open(name string) (fs.File, error) {
	if _, err := sfs.Stat(name); err != nil {
		return nil, err
	}
	return sfs.baseFS.Open(name)
}

var _ fs.StatFS = (*FS)(nil)
