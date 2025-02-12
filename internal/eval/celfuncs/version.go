package celfuncs

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/google/cel-go/cel"
)

func ParseVersion() cel.EnvOption {
	FuncDocs["parseVersion"] = FuncDoc{
		Comment: "Parse a semantic version into major, minor and patch components",
		Args:    []ArgDoc{{"version", ""}},
	}

	return stringReturnsMapErr("parseVersion", func(s string) (map[string]string, error) {
		v, err := semver.NewVersion(s)
		if err != nil {
			return nil, fmt.Errorf("invalid version number: %s", s)
		}
		return map[string]string{
			"major": fmt.Sprint(v.Major()),
			"minor": fmt.Sprint(v.Minor()),
			"patch": fmt.Sprint(v.Patch()),
		}, nil
	})
}
