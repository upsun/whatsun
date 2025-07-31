package dep_test

import (
	_ "embed"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upsun/whatsun/pkg/dep"
)

var (
	//go:embed testdata/python_uv/pyproject_.toml
	pythonPyProject []byte
	//go:embed testdata/python_uv/uv_.lock
	pythonUvLock []byte
)

func TestParsePythonUv(t *testing.T) {
	fsys := fstest.MapFS{
		"pyproject.toml": &fstest.MapFile{Data: pythonPyProject},
		"uv.lock":        &fstest.MapFile{Data: pythonUvLock},
	}

	mgr, err := dep.GetManager(dep.ManagerTypePython, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, mgr.Init())

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"pandas", []dep.Dependency{{
			Name: "pandas", Constraint: ">=2.2.0", Version: "2.3.0", IsDirect: true, ToolName: "uv",
		}}},
		{"numpy", []dep.Dependency{{
			Name: "numpy", Constraint: "==1.26.0", Version: "1.26.0", IsDirect: true, ToolName: "uv",
		}}},
		{"python-dateutil", []dep.Dependency{{
			Name:       "python-dateutil",
			Constraint: ">=2.8.0,<3.0.0",
			Version:    "2.9.0.post0",
			IsDirect:   true,
			ToolName:   "uv",
		}}},
		{"six", []dep.Dependency{{Name: "six", Constraint: ">=1.15.0", Version: "1.17.0", IsDirect: true, ToolName: "uv"}}},
	}
	for _, c := range cases {
		assert.Equal(t, c.dependencies, mgr.Find(c.pattern), c.pattern)
	}

	toGet := []struct {
		name       string
		dependency dep.Dependency
		found      bool
	}{
		{"pandas", dep.Dependency{
			Name: "pandas", Constraint: ">=2.2.0", Version: "2.3.0", IsDirect: true, ToolName: "uv",
		}, true},
		{"numpy", dep.Dependency{
			Name: "numpy", Constraint: "==1.26.0", Version: "1.26.0", IsDirect: true, ToolName: "uv",
		}, true},
		{"python-dateutil", dep.Dependency{
			Name:       "python-dateutil",
			Constraint: ">=2.8.0,<3.0.0",
			Version:    "2.9.0.post0",
			IsDirect:   true,
			ToolName:   "uv",
		}, true},
		{"six", dep.Dependency{Name: "six", Constraint: ">=1.15.0", Version: "1.17.0", IsDirect: true, ToolName: "uv"}, true},
		{"notfound", dep.Dependency{}, false},
	}
	for _, c := range toGet {
		d, ok := mgr.Get(c.name)
		assert.Equal(t, c.found, ok, c.name)
		assert.Equal(t, c.dependency, d, c.name)
	}
}
