package celfuncs

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/google/cel-go/cel"
)

// VersionParse defines a CEL function to split a semantic version into major, minor and patch keys.
func VersionParse() cel.EnvOption {
	FuncComments["version.parse"] = "Parse a semantic version into major, minor and patch components"

	return stringReturnsMapErr("version.parse", func(s string) (map[string]string, error) {

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
