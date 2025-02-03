package celfuncs

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/google/cel-go/cel"
)

// VersionParse defines a CEL function to split a semantic version into major, minor and patch keys.
func VersionParse() cel.EnvOption {
	return stringReturnsMapErr("version.Parse", func(s string) (map[string]string, error) {
		v, err := semver.NewVersion(s)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"major": fmt.Sprint(v.Major()),
			"minor": fmt.Sprint(v.Minor()),
			"patch": fmt.Sprint(v.Patch()),
		}, nil
	})
}
