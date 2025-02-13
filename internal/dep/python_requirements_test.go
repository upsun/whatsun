package dep_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/internal/dep"
)

func TestParseRequirementsTXT(t *testing.T) {
	fsys := fstest.MapFS{
		"requirements.txt": {Data: []byte(`
requests>=2.25.1
numpy==1.21.0
# Commented line
pandas!=1.3.0`)},
	}

	m, err := dep.GetManager(dep.ManagerTypePython, fsys, ".")
	require.NoError(t, err)

	require.NoError(t, m.Init())

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
		assert.Equal(t, c.dependencies, m.Find(c.pattern))
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
