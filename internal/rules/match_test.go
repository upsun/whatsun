package rules_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"what/internal/rules"
)

func TestMatch(t *testing.T) {
	ruleMap := map[string]rules.Rule{
		"a":   {When: "a", Then: "a", Group: "g"},
		"aaa": {When: "aaa", Then: "a"},
		"ab":  {When: "ab", Maybe: []string{"a", "b"}, Group: "g"},
		"bc":  {When: "bc", Maybe: []string{"b", "c"}},
	}

	cases := []struct {
		name        string
		data        []string
		expect      []rules.Match
		expectError bool
	}{
		{
			name: "then_direct",
			data: []string{"a", "x", "y"},
			expect: []rules.Match{
				{Result: "a", Sure: true, Rules: []string{"a"}},
			},
		},
		{
			name: "not_found",
			data: []string{"x", "y", "z"},
		},
		{
			name: "combine_then_maybe_grouped",
			data: []string{"a", "ab"},
			expect: []rules.Match{
				{Result: "a", Sure: true, Rules: []string{"a"}},
			},
		},
		{
			name: "combine_then_maybe_no_group",
			data: []string{"aaa", "bc"},
			expect: []rules.Match{
				{Result: "a", Sure: true, Rules: []string{"aaa"}},
				{Result: "b", Rules: []string{"bc"}},
				{Result: "c", Rules: []string{"bc"}},
			},
		},
	}

	m := rules.Matcher{Rules: ruleMap}

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
