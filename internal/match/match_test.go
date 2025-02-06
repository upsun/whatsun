package match_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"what/internal/match"
)

func TestMatch(t *testing.T) {
	rules := []match.Rule{
		{When: "a", Then: "a"},
		{When: "aaa", Then: "a", Not: []string{"b", "c"}},
		{When: "ab", Maybe: []string{"a", "b"}},
		{When: "bc", Maybe: []string{"b", "c"}},
	}

	cases := []struct {
		name        string
		data        []string
		expect      []match.Match
		expectError bool
	}{
		{
			name: "is_direct",
			data: []string{"a", "x", "y"},
			expect: []match.Match{
				{Result: "a", Sure: true, Report: []string{"a"}},
			},
		},
		{
			name: "not_found",
			data: []string{"x", "y", "z"},
		},
		{
			name: "combine_is_maybe",
			data: []string{"a", "ab", "bc"},
			expect: []match.Match{
				{Result: "a", Sure: true, Report: []string{"a", "ab"}},
				{Result: "b", Sure: false, Report: []string{"ab", "bc"}},
				{Result: "c", Sure: false, Report: []string{"bc"}},
			},
		},
		{
			name: "combine_is_not_maybe",
			data: []string{"aaa", "bc"},
			expect: []match.Match{
				{Result: "a", Sure: true, Report: []string{"aaa"}},
			},
		},
	}

	m := match.Matcher{Rules: rules}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			matches, err := m.Match(func(condition string) (bool, error) {
				return slices.Contains(c.data, condition), nil
			})
			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, c.expect, matches)
			}
		})
	}
}
