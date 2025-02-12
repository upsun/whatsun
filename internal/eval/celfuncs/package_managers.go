package celfuncs

import (
	"sync"

	"github.com/google/cel-go/cel"

	"what/internal/dep"
)

// AllPackageManagerFunctions returns CEL functions for reading package manager dependencies in an fs.FS filesystem.
// This can only be used alongside the FilesystemVariable option.
func AllPackageManagerFunctions() []cel.EnvOption {
	return []cel.EnvOption{
		DepExists(),
		DepVersion(),
	}
}

func DepExists() cel.EnvOption {
	FuncDocs["depExists"] = FuncDoc{
		Comment:     "Check if a project has a dependency",
		Description: "This supports a few package management tools: more may be added later.",
		Args: []ArgDoc{
			{"fs", "The filesystem wrapper"},
			{"managerType", "The manager type (`go`, `js` or `php`)"},
			{"pattern", "The dependency name, accepting `*` as a wildcard"},
		},
	}
	return fsStringStringReturnsBoolErr("depExists", func(fsWrapper filesystemWrapper, manager, pattern string) (bool, error) {
		return depExists(manager, fsWrapper, pattern)
	})
}

func DepVersion() cel.EnvOption {
	FuncDocs["depVersion"] = FuncDoc{
		Comment:     "Find the version of a project dependency",
		Description: "This returns an empty string if the dependency is not found.",
		Args: []ArgDoc{
			{"fs", "The filesystem wrapper"},
			{"managerType", "The manager type (`go`, `js` or `php`)"},
			{"name", "The dependency name"},
		},
	}

	return fsStringStringReturnsStringErr("depVersion", func(fsWrapper filesystemWrapper, manager, name string) (string, error) {
		return depVersion(manager, fsWrapper, name)
	})
}

type managerCacheKey struct {
	managerType string
	fsID        uintptr
	path        string
}

var managerCache struct {
	cache map[managerCacheKey]dep.Manager
	mu    sync.Mutex
}

func depGetCachedManager(managerType string, fsWrapper filesystemWrapper) (dep.Manager, error) {
	managerCache.mu.Lock()
	defer managerCache.mu.Unlock()
	cacheKey := managerCacheKey{managerType: managerType, fsID: fsWrapper.ID, path: fsWrapper.Path}
	if manager, ok := managerCache.cache[cacheKey]; ok {
		return manager, nil
	}
	m, err := dep.GetManager(managerType, fsWrapper.FS, fsWrapper.Path)
	if err != nil {
		return nil, err
	}
	if managerCache.cache == nil {
		managerCache.cache = make(map[managerCacheKey]dep.Manager)
	}
	managerCache.cache[cacheKey] = m
	return m, nil
}

func depVersion(managerType string, fsWrapper filesystemWrapper, name string) (string, error) {
	m, err := depGetCachedManager(managerType, fsWrapper)
	if err != nil {
		return "", err
	}
	d, _ := m.Get(name)
	return d.Version, nil
}

func depExists(managerType string, fsWrapper filesystemWrapper, pattern string) (bool, error) {
	m, err := depGetCachedManager(managerType, fsWrapper)
	if err != nil {
		return false, err
	}
	deps, err := m.Find(pattern)
	if err != nil {
		return false, err
	}
	return len(deps) > 0, nil
}
