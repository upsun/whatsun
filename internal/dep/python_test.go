package dep_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/internal/dep"
)

func TestParseRequirementsTXT(t *testing.T) {
	var fsys fs.FS = &fstest.MapFS{
		"requirements.txt": {Data: []byte(`
			requests>=2.25.1
			numpy==1.21.0
			# Commented line
			pandas!=1.3.0
		`)},
	}

	m, err := dep.GetManager(dep.ManagerTypePython, &fsys, ".")
	require.NoError(t, err)

	toFind := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"requests", []dep.Dependency{{Name: "requests", Constraint: ">=2.25.1"}}},
		{"numpy", []dep.Dependency{{Name: "numpy", Constraint: "==1.21.0"}}},
		{"p*ndas", []dep.Dependency{{Name: "pandas", Constraint: "!=1.3.0"}}},
		{"flask", nil},
	}
	for _, c := range toFind {
		deps, err := m.Find(c.pattern)
		require.NoError(t, err)
		assert.Equal(t, c.dependencies, deps)
	}

	toGet := []struct {
		name       string
		dependency dep.Dependency
		found      bool
	}{
		{"requests", dep.Dependency{Name: "requests", Constraint: ">=2.25.1"}, true},
		{"numpy", dep.Dependency{Name: "numpy", Constraint: "==1.21.0"}, true},
		{"flask", dep.Dependency{}, false},
	}
	for _, c := range toGet {
		d, ok := m.Get(c.name)
		assert.Equal(t, c.found, ok, c.name)
		assert.Equal(t, c.dependency, d, c.name)
	}
}

func TestParsePipfile(t *testing.T) {
	var fsys fs.FS = &fstest.MapFS{
		"Pipfile": {Data: []byte(`
			[packages]
			"requests" = ">=2.25.1"
			numpy = "==1.21.0"
		`)},
	}

	m, err := dep.GetManager(dep.ManagerTypePython, &fsys, ".")
	require.NoError(t, err)

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"requests", []dep.Dependency{{Name: "requests", Constraint: ">=2.25.1"}}},
		{"numpy", []dep.Dependency{{Name: "numpy", Constraint: "==1.21.0"}}},
	}
	for _, c := range cases {
		deps, err := m.Find(c.pattern)
		require.NoError(t, err)
		assert.Equal(t, c.dependencies, deps)
	}
}

func TestParsePyprojectTOML(t *testing.T) {
	var fsys fs.FS = &fstest.MapFS{
		"pyproject.toml": {Data: []byte(`
			[project]
			dependencies = ["requests>=2.25.1", "numpy==1.21.0"]

			[tool.poetry.dependencies]
			tensorflow = "^2.6.0"
			pydantic = {extras = ["mypy"], version = "^2.10.5"}
		`)},
	}

	m, err := dep.GetManager(dep.ManagerTypePython, &fsys, ".")
	require.NoError(t, err)

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"requests", []dep.Dependency{{Name: "requests", Constraint: ">=2.25.1"}}},
		{"numpy", []dep.Dependency{{Name: "numpy", Constraint: "==1.21.0"}}},
		{"tensorflow", []dep.Dependency{{Name: "tensorflow", Constraint: "^2.6.0"}}},
		{"pydantic", []dep.Dependency{{Name: "pydantic", Constraint: "^2.10.5"}}},
	}
	for _, c := range cases {
		deps, err := m.Find(c.pattern)
		require.NoError(t, err)
		assert.Equal(t, c.dependencies, deps)
	}
}
