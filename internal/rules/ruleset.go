package rules

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"gopkg.in/yaml.v3"

	"what/internal/fsgitignore"
)

type Ruleset struct {
	Depends []string         `yaml:"depends"`
	Rules   map[string]*Rule `yaml:"rules"`
}

type Rule struct {
	Name string `yaml:"name"`

	When  string           `yaml:"when"`
	Then  yamlListOrString `yaml:"then"`
	Maybe yamlListOrString `yaml:"maybe"`

	With map[string]string `yaml:"with"`

	GroupList yamlListOrString `yaml:"group"`

	Ignore yamlListOrString `yaml:"ignore"`

	matcher   gitignore.Matcher
	matchInit sync.Once
}

func (r *Rule) IgnoresDirectory(path []string) bool {
	if r.Ignore == nil {
		return false
	}
	r.matchInit.Do(func() {
		r.matcher = gitignore.NewMatcher(fsgitignore.ParsePatterns(r.Ignore, []string{}))
	})
	return r.matcher.Match(path, true)
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

// ParseFiles loads all YAML files in a directory and parses rulesets from them.
func ParseFiles(fsys fs.FS, path string, dest map[string]*Ruleset) error {
	entries, err := fs.ReadDir(fsys, path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		subConfig := make(map[string]*Ruleset)
		f, err := fsys.Open(filepath.Join(path, entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to open config file %s: %w", entry.Name(), err)
		}
		if err := yaml.NewDecoder(f).Decode(&subConfig); err != nil {
			return fmt.Errorf("failed to parse config file %s: %w", entry.Name(), err)
		}
		for k, v := range subConfig {
			dest[k] = v
			// Copy the name to the rule.
			for name, rule := range v.Rules {
				rule.Name = name
				dest[k].Rules[name] = rule
			}
		}
	}
	return nil
}
