package dep_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/pkg/dep"
)

func TestRuby(t *testing.T) {
	fsys := fstest.MapFS{
		"Gemfile": {
			Data: []byte(`gem 'rails', '~> 6.1'
gem "puma"
gem 'nokogiri', '>= 1.10', '< 2.0'`),
		},
		"Gemfile.lock": {
			Data: []byte(`    rails (6.1.4.1)
    puma (5.3.2)
    nokogiri (1.11.3)`),
		},
	}

	m, err := dep.GetManager(dep.ManagerTypeRuby, fsys, ".")
	require.NoError(t, err)

	require.NoError(t, m.Init())

	toFind := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"jekyll", nil},
		{"rails", []dep.Dependency{{
			Name:       "rails",
			Version:    "6.1.4.1",
			Constraint: "~> 6.1",
		}}},
	}
	for _, c := range toFind {
		deps := m.Find(c.pattern)
		assert.Equal(t, c.dependencies, deps)
	}

	toGet := []struct {
		name       string
		dependency dep.Dependency
		found      bool
	}{
		{name: "jekyll"},
		{name: "rails", dependency: dep.Dependency{
			Name:       "rails",
			Version:    "6.1.4.1",
			Constraint: "~> 6.1",
		}, found: true},
	}
	for _, c := range toGet {
		d, ok := m.Get(c.name)
		assert.Equal(t, c.found, ok)
		assert.Equal(t, c.dependency, d)
	}
}
