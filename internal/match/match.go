// Package matcher applies rules to find information in data.
package match

import (
	"fmt"
	"runtime"
	"sort"

	"golang.org/x/sync/errgroup"
)

// Matcher is the matching engine.
//
// Given a set of Rule objects (perhaps defined in YAML), then the Matcher.Match
// method will evaluate the Rule.When condition of each, and combine the
// matched rules into a list of Match results.
//
// For each Match, it can use the Report function to convert the set of matching
// rules into a useful summary (this defaults to a list of conditions, via
// DefaultReportFunc).
type Matcher struct {
	Rules  []Rule
	Report func([]*Rule) any
}

func (f *Matcher) Match(eval func(condition string) (bool, error)) ([]Match, error) {
	if f.Report == nil {
		f.Report = DefaultReportFunc
	}

	eg := errgroup.Group{}
	eg.SetLimit(4 * runtime.GOMAXPROCS(0))
	var s store
	for _, rule := range f.Rules {
		eg.Go(func() error {
			match, err := eval(rule.When)
			if err != nil {
				return err
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
