package rules

import (
	"fmt"
)

// FindMatches will evaluate a list of rules and return a list of Match results.
func FindMatches(rules []RuleSpec, eval func(RuleSpec) (bool, error)) ([]Match, error) {
	var s store
	for _, rule := range rules {
		match, err := eval(rule)
		if err != nil {
			return nil, fmt.Errorf("failed to eval rule %s, condition `%s`: %w", rule.GetName(), rule.GetCondition(), err)
		}
		if match {
			s.Add(rule)
		}
	}

	return s.List()
}

type Match struct {
	Result string
	Sure   bool
	Err    error
	Rules  []RuleSpec
}
