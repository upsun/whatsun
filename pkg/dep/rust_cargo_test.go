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

//go:embed testdata/rust_cargo/Cargo_.toml
var testCargoTOML []byte

//go:embed testdata/rust_cargo/Cargo_.lock
var testCargoLock []byte

func TestParseCargoTOMLAndLock(t *testing.T) {
	// This example was generated with: `cargo add bitflags log rand`
	fsys := fstest.MapFS{
		"Cargo.toml": {Data: testCargoTOML},
		"Cargo.lock": {Data: testCargoLock},
	}

	m, err := dep.GetManager(dep.ManagerTypeRust, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	cases := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"rocket", []dep.Dependency{{
			Name:       "rocket",
			Constraint: "0.5.1",
			Version:    "0.5.1",
		}}},
		{"serde", []dep.Dependency{{
			Name:    "serde",
			Version: "1.0.217",
		}}},
		{"rand*", []dep.Dependency{
			{Name: "rand", Version: "0.9.0", Constraint: "0.9.0"},
			{Name: "rand_chacha", Version: "0.9.0"},
			{Name: "rand_core", Version: "0.9.0"},
		}},
	}
	for _, c := range cases {
		deps := m.Find(c.pattern)
		slices.SortFunc(deps, func(a, b dep.Dependency) int {
			return strings.Compare(a.Name, b.Name)
		})
		assert.Equal(t, c.dependencies, deps)
	}
}
