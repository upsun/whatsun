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

func init() {
	if err := yaml.Unmarshal(configData, &config); err != nil {
		panic(err)
	}
}
