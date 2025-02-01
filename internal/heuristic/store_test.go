package heuristic_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"what/internal/heuristic"
)

func TestHeuristics(t *testing.T) {
	defs := map[string]heuristic.Definition{
		"a":   {Is: "a"},
		"aaa": {Is: "a", Not: []string{"b", "c"}},
		"ab":  {Maybe: []string{"a", "b"}},
		"bc":  {Maybe: []string{"b", "c"}},
	}

	cases := []struct {
		name           string
		sources        []string
		expectFindings []heuristic.Finding
		expectError    bool
	}{
		{
			name:    "is_direct",
			sources: []string{"a", "x", "y"},
			expectFindings: []heuristic.Finding{
				{"a", []string{"a"}},
			},
		},
		{
			name:    "not_found",
			sources: []string{"x", "y", "z"},
		},
		{
			name:    "combine_is_maybe",
			sources: []string{"a", "ab", "bc"},
			expectFindings: []heuristic.Finding{
				{"a", []string{"a", "ab"}},
				{"b", []string{"ab", "bc"}},
				{"c", []string{"bc"}},
			},
		},
		{
			name:    "combine_is_not_maybe",
			sources: []string{"aaa", "bc"},
			expectFindings: []heuristic.Finding{
				{"a", []string{"aaa"}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var store heuristic.Store
			for src, def := range defs {
				if slices.Contains(c.sources, src) {
					store.Add(&def, src)
				}
			}
			findings, err := store.List()
			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, c.expectFindings, findings)
			}
		})
	}
}
