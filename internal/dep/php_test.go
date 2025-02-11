package dep_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/internal/dep"
)

func TestPHP(t *testing.T) {
	fs := fstest.MapFS{
		"composer.json": {Data: []byte(`{
    		"name": "test/test",
			"require": {
		        "php": ">=8.2",
				"symfony/framework-bundle": "^7.2"
			}
		}`)},
		"composer.lock": {Data: []byte(`{
    		"packages": [
				{
					"name": "symfony/framework-bundle",
            		"version": "v7.2.3"
				}
			]
		}`)},
	}

	m, err := dep.GetManager(dep.ManagerTypePHP, fs, ".")
	require.NoError(t, err)

	toFind := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"symfony/*", []dep.Dependency{{
			Vendor:     "symfony",
			Name:       "symfony/framework-bundle",
			Constraint: "^7.2",
			Version:    "v7.2.3",
		}}},
	}
	for _, c := range toFind {
		deps, err := m.Find(c.pattern)
		require.NoError(t, err)
		assert.Equal(t, c.dependencies, deps)
	}

	toGet := []struct {
		name       string
		dependency dep.Dependency
		found      bool
	}{
		{name: "symfony/*"},
		{name: "symfony/framework-bundle", dependency: dep.Dependency{
			Vendor:     "symfony",
			Name:       "symfony/framework-bundle",
			Constraint: "^7.2",
			Version:    "v7.2.3",
		}, found: true},
		{name: "php", dependency: dep.Dependency{
			Name:       "php",
			Constraint: ">=8.2",
		}, found: true},
	}
	for _, c := range toGet {
		d, ok := m.Get(c.name)
		assert.Equal(t, c.found, ok)
		assert.Equal(t, c.dependency, d)
	}
}
