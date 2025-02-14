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

	MaxDepth int `yaml:"max_depth"`
}

type Rule struct {
	Name string `yaml:"name"`

	When  string   `yaml:"when"`
	Then  string   `yaml:"then"`
	Not   []string `yaml:"not"`
	Maybe []string `yaml:"maybe"`

	With map[string]string `yaml:"with"`

	Group     string `yaml:"group"`
	Exclusive bool   `yaml:"exclusive"`
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
