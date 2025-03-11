package dep_test

import (
	_ "embed"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upsun/whatsun/pkg/dep"
)

//go:embed testdata/go_mod/go_.mod
var testGoMod []byte

//go:embed testdata/go_mod/go_.sum
var testGoSum []byte

func TestGoModules(t *testing.T) {
	fsys := fstest.MapFS{
		"go.mod": {Data: testGoMod},
		"go.sum": {Data: testGoSum},
	}

	m, err := dep.GetManager(dep.ManagerTypeGo, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	toFind := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"github.com/gofiber/fiber*", []dep.Dependency{{
			Name:    "github.com/gofiber/fiber/v2",
			Version: "v2.52.6",
		}}},
	}
	for _, c := range toFind {
		assert.Equal(t, c.dependencies, m.Find(c.pattern))
	}

	toGet := []struct {
		name       string
		dependency dep.Dependency
		found      bool
	}{
		{"github.com/gofiber/fiber/v2", dep.Dependency{
			Name:    "github.com/gofiber/fiber/v2",
			Version: "v2.52.6",
		}, true},
	}
	for _, c := range toGet {
		d, ok := m.Get(c.name)
		assert.Equal(t, c.found, ok, c.name)
		assert.Equal(t, c.dependency, d, c.name)
	}
}
