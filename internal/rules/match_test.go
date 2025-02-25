package rules_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"what/internal/rules"
)

func TestMatch(t *testing.T) {
	ruleMap := map[string]*rules.Rule{
		"a":   {Name: "a", When: "a", Then: []string{"a"}, GroupList: []string{"g"}},
		"aaa": {Name: "aaa", When: "aaa", Then: []string{"a"}},
		"ab":  {Name: "ab", When: "ab", Maybe: []string{"a", "b"}, GroupList: []string{"g"}},
		"bc":  {Name: "bc", When: "bc", Maybe: []string{"b", "c"}},
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
			matches, err := m.Match(func(rule *rules.Rule) (bool, error) {
				return slices.Contains(c.data, rule.When), nil
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
