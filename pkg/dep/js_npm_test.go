package dep_test

import (
	_ "embed"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/pkg/dep"
)

//go:embed testdata/js_npm/package_.json
var testNPMPackageJSON []byte

//go:embed testdata/js_npm/package-lock_.json
var testNPMPackageLock []byte

func TestNPM(t *testing.T) {
	// This example was generated with: `npm install gatsby`
	fsys := fstest.MapFS{
		"package.json":      {Data: testNPMPackageJSON},
		"package-lock.json": {Data: testNPMPackageLock},
	}

	m, err := dep.GetManager(dep.ManagerTypeJavaScript, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"gatsby", []dep.Dependency{{
			Name:       "gatsby",
			Constraint: "^5.14.1",
			Version:    "5.14.1",
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
