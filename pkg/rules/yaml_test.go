package rules

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateYAMLAgainstSchema(t *testing.T) {
	cases := []struct {
		name        string
		yamlContent string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid ruleset with then",
			yamlContent: `test_ruleset:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
`,
			expectError: false,
		},
		{
			name: "valid ruleset with maybe",
			yamlContent: `test_ruleset:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      maybe: [test1, test2]
`,
			expectError: false,
		},
		{
			name: "valid ruleset with all fields",
			yamlContent: `test_ruleset:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
      with:
        version: fs.depVersion("js", "test")
      groups: [js, test]
      ignore: "node_modules"
      read_files: ["package.json"]
`,
			expectError: false,
		},
		{
			name: "valid ruleset with groups (alternative form)",
			yamlContent: `test_ruleset:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
      groups: [js, static]
`,
			expectError: false,
		},
		{
			name: "invalid - missing when",
			yamlContent: `test_ruleset:
  rules:
    test-rule:
      then: test
`,
			expectError: true,
			errorMsg:    "when",
		},
		{
			name: "invalid - missing then and maybe",
			yamlContent: `test_ruleset:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
`,
			expectError: true,
			errorMsg:    "then",
		},
		{
			name: "invalid - empty when",
			yamlContent: `test_ruleset:
  rules:
    test-rule:
      when: ""
      then: test
`,
			expectError: true,
			errorMsg:    "String length must be greater than or equal to 1",
		},
		{
			name: "invalid - additional property",
			yamlContent: `test_ruleset:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
      invalid_field: value
`,
			expectError: true,
			errorMsg:    "Additional property",
		},
		{
			name: "invalid - wrong with field type",
			yamlContent: `test_ruleset:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
      with:
        version: 123
`,
			expectError: true,
			errorMsg:    "Invalid type",
		},
		{
			name: "invalid - no rules",
			yamlContent: `test_ruleset:
  rules: {}
`,
			expectError: true,
			errorMsg:    "Must have at least 1 properties",
		},
		{
			name:        "invalid - no ruleset",
			yamlContent: `{}`,
			expectError: true,
			errorMsg:    "Must have at least 1 properties",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateYAMLAgainstSchema([]byte(tc.yamlContent))
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadFromYAMLDir_WithValidation(t *testing.T) {
	cases := []struct {
		name        string
		files       map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid YAML files",
			files: map[string]string{
				"test1.yml": `test_ruleset1:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
`,
				"test2.yml": `test_ruleset2:
  rules:
    another-rule:
      when: fs.fileExists("other.txt")
      maybe: [maybe1, maybe2]
`,
			},
			expectError: false,
		},
		{
			name: "invalid YAML file",
			files: map[string]string{
				"valid.yml": `test_ruleset1:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
`,
				"invalid.yml": `test_ruleset2:
  rules:
    bad-rule:
      when: fs.fileExists("other.txt")
      then: test
      invalid_field: value
`,
			},
			expectError: true,
			errorMsg:    "validation failed for config file invalid.yml",
		},
		{
			name: "duplicate ruleset names",
			files: map[string]string{
				"test1.yml": `test_ruleset:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
`,
				"test2.yml": `test_ruleset:
  rules:
    another-rule:
      when: fs.fileExists("other.txt")
      then: test
`,
			},
			expectError: true,
			errorMsg:    "duplicate ruleset found",
		},
		{
			name: "invalid ruleset name - starts with uppercase",
			files: map[string]string{
				"invalid.yml": `TestRuleset:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
`,
			},
			expectError: true,
			errorMsg:    "invalid ruleset name",
		},
		{
			name: "invalid ruleset name - starts with underscore",
			files: map[string]string{
				"invalid.yml": `_test_ruleset:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
`,
			},
			expectError: true,
			errorMsg:    "invalid ruleset name",
		},
		{
			name: "invalid ruleset name - ends with underscore",
			files: map[string]string{
				"invalid.yml": `test_ruleset_:
  rules:
    test-rule:
      when: fs.fileExists("test.txt")
      then: test
`,
			},
			expectError: true,
			errorMsg:    "invalid ruleset name",
		},
		{
			name: "invalid rule name - starts with uppercase",
			files: map[string]string{
				"invalid.yml": `test_ruleset:
  rules:
    TestRule:
      when: fs.fileExists("test.txt")
      then: test
`,
			},
			expectError: true,
			errorMsg:    "invalid rule name",
		},
		{
			name: "invalid rule name - ends with dash",
			files: map[string]string{
				"invalid.yml": `test_ruleset:
  rules:
    test-rule-:
      when: fs.fileExists("test.txt")
      then: test
`,
			},
			expectError: true,
			errorMsg:    "invalid rule name",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := make(fstest.MapFS)
			for filename, content := range tc.files {
				fsys[filename] = &fstest.MapFile{Data: []byte(content)}
			}

			_, err := LoadFromYAMLDir(fsys, ".")
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
