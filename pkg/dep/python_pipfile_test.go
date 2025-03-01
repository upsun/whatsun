package dep_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/pkg/dep"
)

func TestParsePipfile(t *testing.T) {
	fsys := fstest.MapFS{
		"Pipfile": {Data: []byte(`
[packages]
"requests" = ">=2.25.1"
numpy = "==1.21.0"`)},
	}

	m, err := dep.GetManager(dep.ManagerTypePython, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"requests", []dep.Dependency{{Name: "requests", Constraint: ">=2.25.1"}}},
		{"numpy", []dep.Dependency{{Name: "numpy", Constraint: "==1.21.0"}}},
	}
	for _, c := range cases {
		assert.Equal(t, c.dependencies, m.Find(c.pattern))
	}
}
