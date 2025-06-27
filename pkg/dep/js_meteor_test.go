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

//go:embed testdata/js_meteor/.meteor/packages
var testMeteorPackages []byte

//go:embed testdata/js_meteor/.meteor/versions
var testMeteorVersions []byte

func TestMeteor(t *testing.T) {
	fsys := fstest.MapFS{
		".meteor/packages": {Data: testMeteorPackages},
		".meteor/versions": {Data: testMeteorVersions},
	}

	m, err := dep.GetManager(dep.ManagerTypeJavaScript, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"meteor-base", []dep.Dependency{{Name: "meteor-base", Version: "1.5.1"}}},
		{"ecmascript", []dep.Dependency{{Name: "ecmascript", Version: "0.16.7"}}},
		{"random", []dep.Dependency{{Name: "random", Version: "1.3.2"}}},
	}
	for _, c := range cases {
		deps := m.Find(c.pattern)
		slices.SortFunc(deps, func(a, b dep.Dependency) int {
			return strings.Compare(a.Name, b.Name)
		})
		assert.Equal(t, c.dependencies, deps)
	}
}
