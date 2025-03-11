package dep_test

import (
	_ "embed"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upsun/whatsun/pkg/dep"
)

//go:embed testdata/js_bun/package_.json
var testBunPackageJSON []byte

//go:embed testdata/js_bun/bun_.lock
var testBunLock []byte

func TestBun(t *testing.T) {
	// This example was generated with: `bun install vue`
	fsys := fstest.MapFS{
		"package.json": {Data: testBunPackageJSON},
		"bun.lock":     {Data: testBunLock},
	}

	m, err := dep.GetManager(dep.ManagerTypeJavaScript, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"vue", []dep.Dependency{{
			Name:       "vue",
			Constraint: "^3.5.13",
			Version:    "3.5.13",
		}}},
	}
	for _, c := range cases {
		deps := m.Find(c.pattern)
		slices.SortFunc(deps, func(a, b dep.Dependency) int {
			return strings.Compare(a.Name, b.Name)
		})
		assert.Equal(t, c.dependencies, deps)
	}
}
