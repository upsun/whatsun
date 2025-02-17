package dep_test

import (
	_ "embed"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"what/internal/dep"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/js_deno/deno_.json
var testDenoJSON []byte

func TestDeno(t *testing.T) {
	// This example was generated with: `deno run -A -r https://fresh.deno.dev`
	fsys := fstest.MapFS{
		"deno.json": {Data: testDenoJSON},
	}

	m, err := dep.GetManager(dep.ManagerTypeJavaScript, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"https://deno.land/x/fresh", []dep.Dependency{{
			Name:    "https://deno.land/x/fresh",
			Version: "1.7.3",
		}}},
		{"*preact", []dep.Dependency{{
			Name:    "https://esm.sh/preact",
			Version: "10.22.0",
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
