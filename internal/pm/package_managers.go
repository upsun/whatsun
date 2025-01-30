package pm

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed package_managers.yml
var configData []byte

var allPMs = map[string]*packageManager{}

type packageManager struct {
	name     string
	category string
}

var config *struct {
	PackageManagers map[string][]string `yaml:"package_managers"`
	FilePatterns    map[string][]string `yaml:"file_patterns"`
}

func init() {
	if err := yaml.Unmarshal(configData, &config); err != nil {
		panic(err)
	}
	for cat, names := range config.PackageManagers {
		for _, name := range names {
			allPMs[name] = &packageManager{name: name, category: cat}
		}
	}
}
