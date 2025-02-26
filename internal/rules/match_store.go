package rules

import (
	"slices"
	"strings"
	"sync"
)

type store struct {
	then  map[string][]RuleSpec
	maybe map[string][]RuleSpec

	mutex sync.RWMutex
}

func (s *store) List() ([]Match, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if len(s.then) == 0 && len(s.maybe) == 0 {
		return nil, nil
	}

	// Validate and combine the lists.
	var matches = make([]Match, len(s.then), len(s.then)+len(s.maybe))
	var groupsWithResults = make(map[string]struct{})

	// Add the "then" values.
	i := 0
	for result, rules := range s.then {
		matches[i] = Match{Result: result, Rules: rules, Sure: true}
		i++
		for _, rule := range rules {
			if rg, ok := rule.(WithGroups); ok {
				for _, g := range rg.GetGroups() {
					groupsWithResults[g] = struct{}{}
				}
			}
		}
	}

	// Add the remaining "maybe" values, if there are no "then" values within the same group.
	for result, rules := range s.maybe {
		if _, exists := s.then[result]; exists {
			continue
		}
		var hasResultByGroup bool
		for _, rule := range rules {
			if rg, ok := rule.(WithGroups); ok {
				for _, g := range rg.GetGroups() {
					if _, ok := groupsWithResults[g]; ok {
						hasResultByGroup = true
						break
					}
				}
			}
		}
		if hasResultByGroup {
			continue
		}
		matches = append(matches, Match{Result: result, Rules: rules})
	}

	// Sort the list for consistent output.
	slices.SortFunc(matches, func(a, b Match) int {
		return strings.Compare(a.Result, b.Result)
	})

	return matches, nil
}

func (s *store) Add(rule RuleSpec) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.then == nil {
		s.then = make(map[string][]RuleSpec)
	}
	for _, v := range rule.GetResults() {
		s.then[v] = append(s.then[v], rule)
	}

	if m, ok := rule.(WithMaybeResults); ok {
		if s.maybe == nil {
			s.maybe = make(map[string][]RuleSpec)
		}
		for _, v := range m.GetMaybeResults() {
			s.maybe[v] = append(s.maybe[v], rule)
		}
	}
}
