package rules

import (
	"slices"
	"strings"
	"sync"
)

type store struct {
	results      map[string][]RuleSpec
	resultGroups map[string]struct{}

	maybe map[string][]RuleSpec

	mutex sync.Mutex
}

func (s *store) List() ([]Match, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(s.results) == 0 && len(s.maybe) == 0 {
		return nil, nil
	}

	// Validate and combine the lists.
	var matches = make([]Match, len(s.results), len(s.results)+len(s.maybe))

	// Add the results.
	i := 0
	for result, rules := range s.results {
		matches[i] = Match{Result: result, Rules: rules}
		i++
	}

	// Add "maybe" values, if there are no actual results within the same group.
	for result, rules := range s.maybe {
		if _, exists := s.results[result]; exists {
			continue
		}
		if s.hasResultForGroups(rules) {
			continue
		}
		matches = append(matches, Match{Result: result, Rules: rules, Maybe: true})
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

	// Save the results and associated groups.
	if results := rule.GetResults(); len(results) > 0 {
		if s.results == nil {
			s.results = make(map[string][]RuleSpec)
		}
		for _, v := range results {
			s.results[v] = append(s.results[v], rule)
		}

		// Save associated groups, if any.
		if rg, ok := rule.(WithGroups); ok {
			if s.resultGroups == nil {
				s.resultGroups = make(map[string]struct{})
			}
			for _, g := range rg.GetGroups() {
				s.resultGroups[g] = struct{}{}
			}
		}
	}

	// Save "maybe" results.
	if m, ok := rule.(WithMaybeResults); ok {
		if s.maybe == nil {
			s.maybe = make(map[string][]RuleSpec)
		}
		for _, v := range m.GetMaybeResults() {
			s.maybe[v] = append(s.maybe[v], rule)
		}
	}
}

func (s *store) hasResultForGroups(rules []RuleSpec) bool {
	if s.resultGroups == nil {
		return false
	}
	for _, rule := range rules {
		if rg, ok := rule.(WithGroups); ok {
			for _, g := range rg.GetGroups() {
				if _, ok := s.resultGroups[g]; ok {
					return true
				}
			}
		}
	}
	return false
}
