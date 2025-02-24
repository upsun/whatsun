package rules

import (
	"slices"
	"sort"
	"strings"
	"sync"
)

type store struct {
	then  map[string][]*Rule
	maybe map[string][]*Rule

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
	var groupsWithThen = make(map[string]struct{})

	// Add the "then" values.
	i := 0
	for result, rules := range s.then {
		matches[i] = Match{Result: result, Rules: ruleNames(rules), Sure: true}
		i++
		for _, rule := range rules {
			for _, g := range rule.GroupList {
				groupsWithThen[g] = struct{}{}
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
			for _, g := range rule.GroupList {
				if _, ok := groupsWithThen[g]; ok {
					hasResultByGroup = true
					break
				}
			}
		}
		if hasResultByGroup {
			continue
		}
		matches = append(matches, Match{Result: result, Rules: ruleNames(rules)})
	}

	// Sort the list for consistent output.
	slices.SortFunc(matches, func(a, b Match) int {
		return strings.Compare(a.Result, b.Result)
	})

	return matches, nil
}

func ruleNames(rules []*Rule) []string {
	names := make([]string, len(rules))
	for i, r := range rules {
		names[i] = r.Name
	}
	sort.Strings(names)
	return names
}

func (s *store) Add(rule *Rule) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.maybe == nil {
		s.maybe = make(map[string][]*Rule)
	}
	for _, v := range rule.Maybe {
		s.maybe[v] = append(s.maybe[v], rule)
	}
	if s.then == nil {
		s.then = make(map[string][]*Rule)
	}
	for _, v := range rule.Then {
		s.then[v] = append(s.then[v], rule)
	}
}
