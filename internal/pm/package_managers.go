package pm

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed package_managers.yml
var configData []byte

type configSchema struct {
	Categories      map[string]string   `yaml:"categories"`
	PackageManagers map[string][]string `yaml:"package_managers"`
	FilePatterns    map[string][]string `yaml:"file_patterns"`
}

var packageManagers = map[string]*PackageManager{}

var config *configSchema

func init() {
	if err := yaml.Unmarshal(configData, &config); err != nil {
		panic(err)
	}
	for cat, names := range config.PackageManagers {
		for _, name := range names {
			packageManagers[name] = &PackageManager{Name: name, Category: cat}
		}
	}
}

type PackageManager struct {
	Name string

	// Category is usually a Language* or Framework* constant.
	Category string
}

func (pm *PackageManager) String() string {
	return pm.Category + "/" + pm.Name
}
