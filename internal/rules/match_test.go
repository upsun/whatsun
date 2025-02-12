package rules_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"what/internal/rules"
)

func TestMatch(t *testing.T) {
	ruleMap := map[string]rules.Rule{
		"a":   {When: "a", Then: "a"},
		"aaa": {When: "aaa", Then: "a", Exclusive: true},
		"ab":  {When: "ab", Maybe: []string{"a", "b"}},
		"bc":  {When: "bc", Maybe: []string{"b", "c"}},
	}

	cases := []struct {
		name        string
		data        []string
		expect      []rules.Match
		expectError bool
	}{
		{
			name: "is_direct",
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
			name: "combine_is_maybe",
			data: []string{"a", "ab", "bc"},
			expect: []rules.Match{
				{Result: "a", Sure: true, Rules: []string{"a", "ab"}},
				{Result: "b", Sure: false, Rules: []string{"ab", "bc"}},
				{Result: "c", Sure: false, Rules: []string{"bc"}},
			},
		},
		{
			name: "combine_is_not_maybe",
			data: []string{"aaa", "bc"},
			expect: []rules.Match{
				{Result: "a", Sure: true, Rules: []string{"aaa"}},
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
