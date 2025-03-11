package rules_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/upsun/whatsun/pkg/rules"
)

func TestMatch(t *testing.T) {
	testRules := []rules.RuleSpec{
		&rules.Rule{Name: "a", When: "a", Then: []string{"a"}, GroupList: []string{"g"}},
		&rules.Rule{Name: "aaa", When: "aaa", Then: []string{"a"}},
		&rules.Rule{Name: "ab", When: "ab", Maybe: []string{"a", "b"}, GroupList: []string{"g"}},
		&rules.Rule{Name: "bc", When: "bc", Maybe: []string{"b", "c"}},
	}

	type matchExpectation struct {
		result    any
		maybe     bool
		ruleNames []string
	}

	cases := []struct {
		name        string
		data        []string
		expect      []matchExpectation
		expectError bool
	}{
		{
			name: "then_direct",
			data: []string{"a", "x", "y"},
			expect: []matchExpectation{
				{result: "a", ruleNames: []string{"a"}},
			},
		},
		{
			name: "not_found",
			data: []string{"x", "y", "z"},
		},
		{
			name: "combine_then_maybe_grouped",
			data: []string{"a", "ab"},
			expect: []matchExpectation{
				{result: "a", ruleNames: []string{"a"}},
			},
		},
		{
			name: "combine_then_maybe_no_group",
			data: []string{"aaa", "bc"},
			expect: []matchExpectation{
				{result: "a", ruleNames: []string{"aaa"}},
				{result: "b", maybe: true, ruleNames: []string{"bc"}},
				{result: "c", maybe: true, ruleNames: []string{"bc"}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			matches, err := rules.FindMatches(testRules, func(r rules.RuleSpec) (bool, error) {
				return slices.Contains(c.data, r.GetCondition()), nil
			})
			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				for i, m := range matches {
					assert.Equal(t, c.expect[i].result, m.Result)
					assert.Equal(t, c.expect[i].maybe, m.Maybe)
					for j, r := range m.Rules {
						assert.Equal(t, c.expect[i].ruleNames[j], r.GetName())
					}
				}
			}
		})
	}
}
