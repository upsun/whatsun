package rules

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

type store struct {
	is    map[string][]*Rule
	maybe map[string][]*Rule
	not   map[string]struct{}

	exclusiveByGroup map[string]string

	mutex sync.RWMutex
}

func (s *store) List(report func(rules []*Rule) any) ([]Match, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if len(s.is) == 0 && len(s.maybe) == 0 {
		return nil, nil
	}

	// Validate and combine the lists.
	var matches = make([]Match, 0, len(s.is)+len(s.maybe))

	// Add the "is" values, checking for conflicts with "not", and merging
	// rules with matching "maybe" values.
	for result, rules := range s.is {
		if _, conflicting := s.not[result]; conflicting {
			return nil, fmt.Errorf("conflict found: %s", result)
		}
		for _, r := range rules {
			if r.Exclusive {
				if conflict, conflicting := s.exclusiveByGroup[r.Group]; conflicting && conflict != result {
					// Report the error as a match, so that conflicts don't fail the whole analysis.
					if r.Group != "" {
						matches = append(matches, Match{
							Err: fmt.Errorf("conflict found in group %s: %s vs %s", r.Group, result, conflict)})
					} else {
						matches = append(matches, Match{
							Err: fmt.Errorf("conflict found: %s", result)})
					}
					continue
				}
			}
		}
		if m, ok := s.maybe[result]; ok {
			rules = append(rules, m...)
		}
		matches = append(matches, Match{Result: result, Report: report(rules), Sure: true})
	}

	// Add the remaining "maybe" values.
	for result, rules := range s.maybe {
		if _, exists := s.is[result]; exists {
			continue
		}
		if _, conflicting := s.not[result]; conflicting {
			continue
		}
		var hasConflict bool
		for _, r := range rules {
			if conflict, conflicting := s.exclusiveByGroup[r.Group]; conflicting && conflict != result {
				hasConflict = true
				break
			}
		}
		if !hasConflict {
			matches = append(matches, Match{Result: result, Report: report(rules)})
		}
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
			s.maybe[v] = append(s.maybe[v], rule)
		}
	}

	if rule.Then != "" {
		if s.is == nil {
			s.is = make(map[string][]*Rule)
		}
		if rule.Exclusive {
			if s.exclusiveByGroup == nil {
				s.exclusiveByGroup = make(map[string]string)
			}
			s.exclusiveByGroup[rule.Group] = rule.Then
		}
		s.is[rule.Then] = append(s.is[rule.Then], rule)
	}
}
