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

//go:embed testdata/js_pnpm/package_.json
var testPNPMPackageJSON []byte

//go:embed testdata/js_pnpm/pnpm-lock_.yaml
var testPNPMLock []byte

func TestPNPM(t *testing.T) {
	// This example was generated with: `npm install gatsby`
	fsys := fstest.MapFS{
		"package.json":   {Data: testPNPMPackageJSON},
		"pnpm-lock.yaml": {Data: testPNPMLock},
	}

	m, err := dep.GetManager(dep.ManagerTypeJavaScript, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"strapi", []dep.Dependency{{
			Name:       "strapi",
			Constraint: "^3.6.11",
			Version:    "3.6.11",
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
