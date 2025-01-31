package pm

import (
	_ "embed"

	"gopkg.in/yaml.v3"

	"what/internal/heuristic"
)

//go:embed package_managers.yml
var configData []byte

var config *struct {
	FilePatterns map[string]*heuristic.Definition `yaml:"file_patterns"`
}

func init() {
	if err := yaml.Unmarshal(configData, &config); err != nil {
		panic(err)
	}
}
