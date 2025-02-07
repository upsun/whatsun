// Package matcher applies rules to find information in data.
package match

import (
	"fmt"
	"golang.org/x/sync/errgroup"
	"runtime"
	"sort"
	"what"
)

// Matcher is the matching engine.
//
// Given a set of what.Rule objects (perhaps defined in YAML), then the
// Matcher.Match method will evaluate the Rule.When condition of each, and
// combine the matched rules into a list of Match results.
//
// For each Match, it can use the Report function to convert the set of matching
// rules into a useful summary (this defaults to a list of conditions, via
// DefaultReportFunc).
type Matcher struct {
	Rules  []what.Rule
	Report func([]*what.Rule) any
}

func (f *Matcher) Match(eval func(condition string) (bool, error)) ([]Match, error) {
	if f.Report == nil {
		f.Report = DefaultReportFunc
	}

	eg := errgroup.Group{}
	eg.SetLimit(runtime.GOMAXPROCS(0))
	var s store
	for _, rule := range f.Rules {
		eg.Go(func() error {
			match, err := eval(rule.When)
			if err != nil {
				return fmt.Errorf("when evaluating condition `%s`: %w", rule.When, err)
			}
			if match {
				s.Add(&rule)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return s.List(f.Report)
}

type Match struct {
	Result string
	Sure   bool
	Report any
}

func (m *Match) String() string {
	return fmt.Sprintf("%s (report: %v)", m.Result, m.Report)
}

func DefaultReportFunc(rules []*what.Rule) any {
	report := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule.Name != "" {
			report = append(report, rule.Name)
		} else if rule.When != "" {
			report = append(report, "when: "+rule.When)
		}
	}
	sort.Strings(report)
	return report
}
