package celfuncs

import (
	"io/fs"

	"github.com/google/cel-go/cel"

	"what/internal/dep"
)

// AllPackageManagerFunctions returns CEL functions for reading package manager dependencies in an fs.FS filesystem.
func AllPackageManagerFunctions(fsys *fs.FS, root *string) []cel.EnvOption {
	if root == nil {
		dot := "."
		root = &dot
	}
	return []cel.EnvOption{
		DepHas(fsys, root),
		DepGetVersion(fsys, root),
	}
}

func DepHas(fsys *fs.FS, root *string) cel.EnvOption {
	FuncDocs["dep.has"] = FuncDoc{
		Comment:     "Check if a project has a dependency",
		Description: "This supports a few package management tools: more may be added later.",
		Args: []ArgDoc{
			{"managerType", "The manager type (`go`, `js` or `php`)"},
			{"pattern", "The dependency name, accepting `*` as a wildcard"},
		},
	}
	return stringStringReturnsBoolErr("dep.has", func(manager, pattern string) (bool, error) {
		return depExists(manager, fsys, *root, pattern)
	})
}

func DepGetVersion(fsys *fs.FS, root *string) cel.EnvOption {
	FuncDocs["dep.getVersion"] = FuncDoc{
		Comment:     "Find the version of a project dependency",
		Description: "This returns an empty string if the dependency is not found.",
		Args: []ArgDoc{
			{"managerType", "The manager type (`go`, `js` or `php`)"},
			{"name", "The dependency name"},
		},
	}

	return stringStringReturnsStringErr("dep.getVersion", func(manager, name string) (string, error) {
		return depVersion(manager, fsys, *root, name)
	})
}

func depVersion(managerType string, fsys *fs.FS, path, name string) (string, error) {
	m, err := dep.GetManager(managerType, fsys, path)
	if err != nil {
		return "", err
	}
	d, _ := m.Get(name)
	return d.Version, nil
}

func depExists(managerType string, fsys *fs.FS, path, pattern string) (bool, error) {
	m, err := dep.GetManager(managerType, fsys, path)
	if err != nil {
		return false, err
	}
	deps, err := m.Find(pattern)
	if err != nil {
		return false, err
	}
	return len(deps) > 0, nil
}
