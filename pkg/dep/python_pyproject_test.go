package dep_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upsun/whatsun/pkg/dep"
)

func TestParsePyprojectTOML(t *testing.T) {
	fsys := fstest.MapFS{
		"pyproject.toml": {Data: []byte(`
			[project]
			dependencies = ["requests>=2.25.1", "numpy==1.21.0"]

			[tool.poetry.dependencies]
			tensorflow = "^2.6.0"
			pydantic = {extras = ["mypy"], version = "^2.10.5"}
		`)},
	}

	m, err := dep.GetManager(dep.ManagerTypePython, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"requests", []dep.Dependency{{Name: "requests", Constraint: ">=2.25.1", IsDirect: true, ToolName: "poetry"}}},
		{"numpy", []dep.Dependency{{Name: "numpy", Constraint: "==1.21.0", IsDirect: true, ToolName: "poetry"}}},
		{"tensorflow", []dep.Dependency{{Name: "tensorflow", Constraint: "^2.6.0", IsDirect: true, ToolName: "poetry"}}},
		{"pydantic", []dep.Dependency{{Name: "pydantic", Constraint: "^2.10.5", IsDirect: true, ToolName: "poetry"}}},
	}
	for _, c := range cases {
		assert.Equal(t, c.dependencies, m.Find(c.pattern))
	}
}
