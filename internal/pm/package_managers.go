package pm

import (
	_ "embed"

	"gopkg.in/yaml.v3"

	"what/internal/match"
)

//go:embed package_managers.yml
var configData []byte

var config *struct {
	PackageManagers struct {
		Rules []match.Rule `yaml:"rules"`
	} `yaml:"package_managers"`
}

func rules() ([]match.Rule, error) {
	if config == nil {
		if err := yaml.Unmarshal(configData, &config); err != nil {
			return nil, err
		}
	}

	return config.PackageManagers.Rules, nil
}
