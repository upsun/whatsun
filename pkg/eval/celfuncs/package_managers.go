package celfuncs

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"

	"github.com/upsun/whatsun/pkg/dep"
)

// AllPackageManagerFunctions returns CEL functions for reading package manager dependencies in an fs.FS filesystem.
// This can only be used alongside the FilesystemVariables options.
func AllPackageManagerFunctions(docs *Docs) []cel.EnvOption {
	return []cel.EnvOption{
		DepExists(docs),
		DepVersion(docs),
	}
}

func managerTypeComment() string {
	return fmt.Sprintf("The manager type (one of: `%s`)", strings.Join(dep.AllManagerTypes, "`, `"))
}

func DepExists(docs *Docs) cel.EnvOption {
	docs.AddFunction("depExists", FuncDoc{
		Comment:     "Check if a project has a dependency",
		Description: "This supports a few package management tools: more may be added later.",
		Args: []ArgDoc{
			{"fs", "The filesystem wrapper"},
			{"managerType", managerTypeComment()},
			{"pattern", "The dependency name, accepting `*` as a wildcard"},
		},
	})

	return fsBinaryFunction("depExists", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		func(fsWrapper filesystemWrapper, managerType string, pattern string) (bool, error) {
			m, err := dep.GetCachedManager(managerType, fsWrapper)
			if err != nil {
				return false, err
			}
			return len(m.Find(pattern)) > 0, nil
		},
	)
}

func DepVersion(docs *Docs) cel.EnvOption {
	docs.AddFunction("depVersion", FuncDoc{
		Comment:     "Find the version of a project dependency",
		Description: "This returns an empty string if the dependency is not found.",
		Args: []ArgDoc{
			{"fs", "The filesystem wrapper"},
			{"managerType", managerTypeComment()},
			{"name", "The dependency name"},
		},
	})

	return fsBinaryFunction("depVersion", []*cel.Type{cel.StringType, cel.StringType}, cel.StringType,
		func(fsWrapper filesystemWrapper, managerType string, name string) (string, error) {
			m, err := dep.GetCachedManager(managerType, fsWrapper)
			if err != nil {
				return "", err
			}
			d, _ := m.Get(name)
			return d.Version, nil
		},
	)
}
