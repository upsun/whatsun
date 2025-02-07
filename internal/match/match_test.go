package match_test

import (
	"github.com/stretchr/testify/assert"
	"slices"
	"testing"
	"what"

	"what/internal/match"
)

func TestMatch(t *testing.T) {
	rules := []what.Rule{
		{When: "a", Then: "a"},
		{When: "aaa", Then: "a", Exclusive: true},
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
				{Result: "a", Sure: true, Report: []string{"when: a"}},
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
				{Result: "a", Sure: true, Report: []string{"when: a", "when: ab"}},
				{Result: "b", Sure: false, Report: []string{"when: ab", "when: bc"}},
				{Result: "c", Sure: false, Report: []string{"when: bc"}},
			},
		},
		{
			name: "combine_is_not_maybe",
			data: []string{"aaa", "bc"},
			expect: []match.Match{
				{Result: "a", Sure: true, Report: []string{"when: aaa"}},
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
