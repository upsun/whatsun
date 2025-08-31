package rules

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var schemaBytes []byte

// validateYAMLAgainstSchema validates YAML content against the embedded JSON schema
func validateYAMLAgainstSchema(yamlData []byte) error {
	// Convert YAML to JSON for schema validation
	var data any
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	// Load the schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	if !result.Valid() {
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}
		return fmt.Errorf("schema validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

type YAMLListOrString []string

func (l *YAMLListOrString) UnmarshalYAML(unmarshal func(interface{}) error) error {
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

// ValidateName checks if a rule or ruleset name is valid.
var ValidateName = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*[a-z0-9]$`).MatchString

// LoadFromYAMLDir loads all YAML files in a directory and parses rulesets from them.
func LoadFromYAMLDir(fsys fs.FS, path string) ([]RulesetSpec, error) {
	entries, err := fs.ReadDir(fsys, path)
	if err != nil {
		return nil, err
	}
	var setMap = make(map[string]*Ruleset)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		subConfig := make(map[string]struct {
			Rules map[string]*Rule `yaml:"rules"`
		})
		f, err := fsys.Open(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to open config file %s: %w", entry.Name(), err)
		}

		// Read the file content for validation
		yamlData, err := io.ReadAll(f)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to read config file %s: %w", entry.Name(), err)
		}
		f.Close()

		// Validate against schema
		if err := validateYAMLAgainstSchema(yamlData); err != nil {
			return nil, fmt.Errorf("validation failed for config file %s: %w", entry.Name(), err)
		}

		// Parse the YAML
		if err := yaml.Unmarshal(yamlData, &subConfig); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", entry.Name(), err)
		}
		for name, rs := range subConfig {
			if !ValidateName(name) {
				return nil, fmt.Errorf("invalid ruleset name: %s", name)
			}
			if _, ok := setMap[name]; ok {
				return nil, fmt.Errorf("duplicate ruleset found: '%s'", name)
			}
			rules := make([]RuleSpec, len(rs.Rules))
			i := 0
			for k, rule := range rs.Rules {
				if !ValidateName(k) {
					return nil, fmt.Errorf("invalid rule name: %s", k)
				}
				rule.Name = k
				rules[i] = rule
				i++
			}
			setMap[name] = &Ruleset{
				Name:  name,
				Rules: rules,
			}
		}
	}

	var sets = make([]RulesetSpec, len(setMap))
	i := 0
	for _, rs := range setMap {
		sets[i] = rs
		i++
	}

	return sets, nil
}
