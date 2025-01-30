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

func (s *store) list() []DetectedPM {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.byCategory == nil {
		return nil
	}

	// Flatten the list and merge sources.
	uniq := make(map[*packageManager][]string)
	for _, v := range s.byCategory {
		for pm, a := range v {
			if _, ok := uniq[pm]; ok {
				uniq[pm] = append(uniq[pm], a.sources...)
			} else {
				uniq[pm] = a.sources
			}
		}
	}

	// Convert the list to the public type.
	// For consistent output, sort each sources list, and then sort the whole list.
	var pms = make([]DetectedPM, len(uniq))
	i := 0
	for pm, sources := range uniq {
		sort.StringSlice(sources).Sort()
		pms[i] = DetectedPM{Name: pm.name, Category: pm.category, Sources: sources}
		i++
	}
	slices.SortFunc(pms, func(a, b DetectedPM) int {
		return strings.Compare(a.Name, b.Name)
	})

	return pms
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
	inCategory, found := s.byCategory[pm.category]
	if !found {
		s.byCategory[pm.category] = map[*packageManager]attr{
			pm: {sources: []string{src}, certain: certain},
		}
		return nil
	}

	// The same package manager has been found: merge the attributes.
	if existing, ok := inCategory[pm]; ok {
		existing.sources = append(existing.sources, src)
		if certain {
			existing.certain = true
		}
		s.byCategory[pm.category][pm] = existing
		return nil
	}

	// If this one is 'certain', then check if a different package manager is in the
	// same category, and error if it is also 'certain', or remove it.
	if certain {
		for k, a := range inCategory {
			if a.certain {
				return fmt.Errorf(
					"conflicting PMs found for source %s, category %s (%s vs %s)",
					src, pm.category, pm, k)
			}
			delete(inCategory, k)
		}
	}

	return nil
}
