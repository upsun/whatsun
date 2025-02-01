package match

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

type store struct {
	report func([]*Rule) any

	is    map[string][]*Rule
	maybe map[string][]*Rule
	not   map[string]struct{}

	mutex sync.RWMutex
}

func (s *store) List() ([]Match, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if len(s.is) == 0 && len(s.maybe) == 0 {
		return nil, nil
	}

	if s.report == nil {
		s.report = DefaultReportFunc
	}

	// Validate and combine the lists.
	var matches = make([]Match, 0, len(s.is)+len(s.maybe))

	// Add the "is" values, checking for conflicts with "not", and merging
	// rules with matching "maybe" values.
	for result, rules := range s.is {
		if c, conflicting := s.not[result]; conflicting {
			return nil, fmt.Errorf("conflict found: %s vs %s", result, c)
		}
		if m, ok := s.maybe[result]; ok {
			rules = append(rules, m...)
		}
		matches = append(matches, Match{Result: result, Report: s.report(rules)})
	}

	// Add the remaining "maybe" values.
	for result, rules := range s.maybe {
		if _, exists := s.is[result]; exists {
			continue
		}
		if _, conflicting := s.not[result]; conflicting {
			continue
		}
		matches = append(matches, Match{Result: result, Report: s.report(rules)})
	}

	// Sort the list for consistent output.
	slices.SortFunc(matches, func(a, b Match) int {
		return strings.Compare(a.Result, b.Result)
	})

	return matches, nil
}

func (s *store) Add(rule *Rule) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(rule.Not) > 0 {
		if s.not == nil {
			s.not = make(map[string]struct{})
		}
		for _, v := range rule.Not {
			s.not[v] = struct{}{}
		}
	}

	if len(rule.Maybe) > 0 {
		if s.maybe == nil {
			s.maybe = make(map[string][]*Rule)
		}
		for _, v := range rule.Maybe {
			if _, ok := s.maybe[v]; ok {
				s.maybe[v] = append(s.maybe[v], rule)
			} else {
				s.maybe[v] = []*Rule{rule}
			}
		}
	}

	if rule.Then != "" {
		if s.is == nil {
			s.is = make(map[string][]*Rule)
		}
		if _, ok := s.is[rule.Then]; ok {
			s.is[rule.Then] = append(s.is[rule.Then], rule)
		} else {
			s.is[rule.Then] = []*Rule{rule}
		}
	}
}
