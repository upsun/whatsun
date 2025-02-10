package rules

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"what"
)

var Config map[string]Ruleset

type Ruleset struct {
	Depends []string        `yaml:"depends"`
	Rules   map[string]Rule `yaml:"rules"`
}

type Rule struct {
	Name string `yaml:"name"`

	When  string           `yaml:"when"`
	Then  yamlListOrString `yaml:"then"`
	Maybe yamlListOrString `yaml:"maybe"`

	With map[string]string `yaml:"with"`

	GroupList yamlListOrString `yaml:"group"`

	Ignore yamlListOrString `yaml:"ignore"`
}

type yamlListOrString []string

func (l *yamlListOrString) UnmarshalYAML(unmarshal func(interface{}) error) error {
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

func init() {
	if err := parseConfig(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseConfig() error {
	dirname := "config"
	entries, err := fs.ReadDir(what.ConfigData, dirname)
	if err != nil {
		return err
	}
	Config = make(map[string]Ruleset)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") {
			subConfig := make(map[string]Ruleset)
			f, err := what.ConfigData.Open(filepath.Join(dirname, entry.Name()))
			if err != nil {
				return fmt.Errorf("failed to open config file %s: %w", entry.Name(), err)
			}
			if err := yaml.NewDecoder(f).Decode(&subConfig); err != nil {
				return fmt.Errorf("failed to parse config file %s: %w", entry.Name(), err)
			}
			for k, v := range subConfig {
				Config[k] = v
				// Copy the name to the rule.
				for name, rule := range v.Rules {
					rule.Name = name
					Config[k].Rules[name] = rule
				}
			}
		}
	}
	return nil
}
