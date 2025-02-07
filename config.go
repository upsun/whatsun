package what

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed config
var configData embed.FS

var Config map[string]Ruleset

type Ruleset struct {
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

func init() {
	if err := parseConfig(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseConfig() error {
	dirname := "config"
	entries, err := fs.ReadDir(configData, dirname)
	if err != nil {
		return err
	}
	Config = make(map[string]Ruleset)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") {
			subConfig := make(map[string]Ruleset)
			f, err := configData.Open(filepath.Join(dirname, entry.Name()))
			if err != nil {
				return fmt.Errorf("failed to open config file %s: %w", entry.Name(), err)
			}
			if err := yaml.NewDecoder(f).Decode(&subConfig); err != nil {
				return fmt.Errorf("failed to parse config file %s: %w", entry.Name(), err)
			}
			for k, v := range subConfig {
				Config[k] = v
			}
		}
	}
	return nil
}
