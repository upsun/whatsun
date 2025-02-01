package heuristic

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
)

type Store struct {
	is    map[string]sourceList
	maybe map[string]sourceList
	not   map[string]struct{}
	mutex sync.RWMutex
}

type sourceList = []string

func (s *Store) List() ([]Finding, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if len(s.is) == 0 && len(s.maybe) == 0 {
		return nil, nil
	}

	// Validate and combine the lists.
	// For consistent output, sort each sources list, and then sort the whole list.
	var findings = make([]Finding, 0, len(s.is)+len(s.maybe))

	// Add the "is" values, checking for conflicts with "not", and merging
	// sources with matching "maybe" values.
	for name, srcs := range s.is {
		if c, conflicting := s.not[name]; conflicting {
			return nil, fmt.Errorf("conflict found: %s vs %s", name, c)
		}
		if m, ok := s.maybe[name]; ok {
			srcs = append(srcs, m...)
		}
		sort.Strings(srcs)
		findings = append(findings, Finding{Name: name, Sources: srcs})
	}

	// Add the remaining "maybe" values.
	for name, srcs := range s.maybe {
		if _, exists := s.is[name]; exists {
			continue
		}
		if _, conflicting := s.not[name]; conflicting {
			continue
		}
		sort.Strings(srcs)
		findings = append(findings, Finding{Name: name, Sources: srcs})
	}

	slices.SortFunc(findings, func(a, b Finding) int {
		return strings.Compare(a.Name, b.Name)
	})

	return findings, nil
}

func (s *Store) Add(def *Definition, sources ...string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(def.Not) > 0 {
		if s.not == nil {
			s.not = make(map[string]struct{})
		}
		for _, v := range def.Not {
			s.not[v] = struct{}{}
		}
	}

	if len(def.Maybe) > 0 {
		if s.maybe == nil {
			s.maybe = make(map[string]sourceList)
		}
		for _, v := range def.Maybe {
			if _, ok := s.maybe[v]; ok {
				s.maybe[v] = append(s.maybe[v], sources...)
			} else {
				s.maybe[v] = sources
			}
		}
	}

	if def.Is != "" {
		if s.is == nil {
			s.is = make(map[string]sourceList)
		}
		if _, ok := s.is[def.Is]; ok {
			s.is[def.Is] = append(s.is[def.Is], sources...)
		} else {
			s.is[def.Is] = sources
		}
	}
}
