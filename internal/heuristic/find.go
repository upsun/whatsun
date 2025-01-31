// Package heuristic processes simple heuristics.
//
// Given a set of Definition objects (perhaps defined in YAML), added
// conditionally to a heuristic.Store, it will resolve them (based on "is", "not"
// and "maybe" rules) to produce a list of Finding objects. The Sources in each
// Finding can be used for tracking why each condition passed.
//
// Note: for now, this package does not handle the conditions themselves.
package heuristic

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Finding struct {
	Name    string
	Sources []string
}

func (d *Finding) String() string {
	return fmt.Sprintf("%s (via: %s)", d.Name, d.Sources)
}

type Definition struct {
	Is    string   `yaml:"is"`
	Not   []string `yaml:"not"`
	Maybe []string `yaml:"maybe"`
}

func (d *Definition) UnmarshalYAML(v *yaml.Node) error {
	var str string
	if err := v.Decode(&str); err == nil {
		*d = Definition{Is: str}
		return nil
	}

	type tmpType Definition // alias to avoid recursion
	var tmp tmpType
	if err := v.Decode(&tmp); err != nil {
		return err
	}

	*d = Definition(tmp)
	return nil
}
