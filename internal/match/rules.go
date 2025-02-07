package match

import (
	"io"

	"gopkg.in/yaml.v3"
)

type Config map[string]RuleSet

type RuleSet struct {
	Depends []string `yaml:"depends"`
	Rules   []Rule   `yaml:"rules"`
}

type Rule struct {
	When  string   `yaml:"when"`
	Then  string   `yaml:"then"`
	Not   []string `yaml:"not"`
	Maybe []string `yaml:"maybe"`

	Group     string `yaml:"group"`
	Exclusive bool   `yaml:"exclusive"`
}

func ParseConfig(r io.Reader) (Config, error) {
	c := Config{}
	if err := yaml.NewDecoder(r).Decode(&c); err != nil {
		return nil, err
	}
	return c, nil
}
