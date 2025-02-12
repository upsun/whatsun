package rules

import (
	"fmt"
)

// Matcher is the matching engine.
//
// Given a set of Rule objects (perhaps defined in YAML), then the
// Matcher.Match method will evaluate the Rule.When condition of each, and
// combine the matched rules into a list of Match results.
//
// For each Match, it can use the Report function to convert the set of matching
// rules into a useful summary (this defaults to a list of conditions, via
// DefaultReportFunc).
type Matcher struct {
	Rules map[string]Rule
}

func (f *Matcher) Match(eval func(condition string) (bool, error)) ([]Match, error) {
	var s store
	for name, rule := range f.Rules {
		match, err := eval(rule.When)
		if err != nil {
			return nil, fmt.Errorf("failed to eval rule %s, condition `%s`: %w", name, rule.When, err)
		}
		if match {
			rule.Name = name
			s.Add(&rule)
		}
	}

	return s.List()
}

type Match struct {
	Result string
	Sure   bool
	Err    error
	Rules  []string
}
