package rules

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type YAMLListOrString []string

func (l *YAMLListOrString) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s []string
	if err := unmarshal(&s); err != nil {
		var str string
		err := unmarshal(&str)
		if err != nil {
			return err
		}
		*l = []string{str}
	} else {
		*l = s
	}
	return nil
}

// LoadFromYAMLDir loads all YAML files in a directory and parses rulesets from them.
func LoadFromYAMLDir(fsys fs.FS, path string) ([]RulesetSpec, error) {
	entries, err := fs.ReadDir(fsys, path)
	if err != nil {
		return nil, err
	}
	var setMap = make(map[string]*Ruleset)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		subConfig := make(map[string]struct {
			Name    string           `yaml:"name,omitempty"`
			Depends []string         `yaml:"depends"`
			Rules   map[string]*Rule `yaml:"rules"`
		})
		f, err := fsys.Open(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to open config file %s: %w", entry.Name(), err)
		}
		if err := yaml.NewDecoder(f).Decode(&subConfig); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", entry.Name(), err)
		}
		for name, rs := range subConfig {
			if _, ok := setMap[name]; ok {
				return nil, fmt.Errorf("duplicate ruleset found: %s", name)
			}
			rules := make([]RuleSpec, len(rs.Rules))
			i := 0
			for k, rule := range rs.Rules {
				rule.Name = k
				rules[i] = rule
				i++
			}
			setMap[name] = &Ruleset{
				Name:  rs.Name,
				Rules: rules,
			}
		}
	}

	var sets = make([]RulesetSpec, len(setMap))
	i := 0
	for name, rs := range setMap {
		rs.Name = name
		sets[i] = rs
		i++
	}

	return sets, nil
}
