// Package matcher applies rules to find information in data.
package match

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
)

// Matcher is the matching engine.
//
// Given a set of Rule objects (perhaps defined in YAML), then the Matcher.Match
// method will apply each Rule to given data using the Eval function.
//
// It will produce a list of Match results, using the Report function to convert
// the set of rules that matched into a useful summary (which can be nil).
type Matcher struct {
	Rules  []Rule
	Eval   func(data any, condition any) (bool, error)
	Report func([]*Rule) any
}

func DefaultEvalFunc(data any, condition any) (bool, error) {
	return isEqual(data, condition), nil
}

func DefaultReportFunc(rules []*Rule) any {
	report := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule.When != "" {
			report = append(report, rule.When)
		}
	}
	sort.Strings(report)
	return report
}

func (f *Matcher) Match(data any) ([]Match, error) {
	if f.Eval == nil {
		f.Eval = DefaultEvalFunc
	}

	s := store{report: f.Report}
	for _, rule := range f.Rules {
		match, err := f.Eval(data, rule.When)
		if err != nil {
			return nil, err
		}
		if match {
			s.Add(&rule)
		}
	}

	return s.List()
}

type Match struct {
	Result string
	Report any
}

func (m *Match) String() string {
	return fmt.Sprintf("%s (report: %v)", m.Result, m.Report)
}

type Rule struct {
	When  string   `yaml:"when"`
	Then  string   `yaml:"then"`
	Not   []string `yaml:"not"`
	Maybe []string `yaml:"maybe"`
}

func isEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}

	aB, ok := a.([]byte)
	if !ok {
		return reflect.DeepEqual(a, b)
	}

	bB, ok := b.([]byte)
	if !ok {
		return false
	}
	if aB == nil || bB == nil {
		return aB == nil && bB == nil
	}
	return bytes.Equal(aB, bB)
}
