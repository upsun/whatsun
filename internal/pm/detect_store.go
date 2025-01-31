package pm

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
)

type store struct {
	byCategory map[string]map[*packageManager]attr
	mutex      sync.RWMutex
}

type attr struct {
	sources []string
	certain bool
}

func (s *store) list() ([]DetectedPM, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.byCategory == nil {
		return nil, nil
	}

	// Validate and flatten the list.
	// For consistent output, sort each sources list, and then sort the whole list.
	var pms []DetectedPM
	for cat, v := range s.byCategory {
		var certain *packageManager
		for pm, a := range v {
			if a.certain && certain == nil {
				certain = pm
			}
		}
		for pm, a := range v {
			if certain != nil && certain != pm {
				if a.certain {
					return nil, fmt.Errorf("conflicting package managers found for category %s", cat)
				}
				continue
			}
			sort.StringSlice(a.sources).Sort()
			pms = append(pms, DetectedPM{Name: pm.name, Category: pm.category, Sources: a.sources})
		}
	}

	slices.SortFunc(pms, func(a, b DetectedPM) int {
		return strings.Compare(a.Name, b.Name)
	})

	return pms, nil
}

func (s *store) add(name string, src string, certain bool) error {
	pm, ok := allPMs[name]
	if !ok {
		return fmt.Errorf("no package manager found for: %s", name)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.byCategory == nil {
		s.byCategory = make(map[string]map[*packageManager]attr)
	}

	// Nothing is found in this category yet: simply add this one.
	if _, found := s.byCategory[pm.category]; !found {
		s.byCategory[pm.category] = map[*packageManager]attr{
			pm: {sources: []string{src}, certain: certain},
		}
		return nil
	}

	// If the same package manager is found, merge the attributes.
	if existing, ok := s.byCategory[pm.category][pm]; ok {
		s.byCategory[pm.category][pm] = attr{
			sources: append(existing.sources, src),
			certain: certain || existing.certain,
		}
		return nil
	}

	s.byCategory[pm.category][pm] = attr{sources: []string{src}, certain: certain}

	return nil
}
