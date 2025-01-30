package pm

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
)

type detectionStore struct {
	byCategory map[string]map[*PackageManager]detectedAttributes
	mutex      sync.RWMutex
}

type detectedAttributes struct {
	sources []string
	certain bool
}

func (s *detectionStore) list() []DetectedPM {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.byCategory == nil {
		return nil
	}
	uniq := make(map[*PackageManager][]string)
	for _, pmAttrs := range s.byCategory {
		for pm, attrs := range pmAttrs {
			if _, ok := uniq[pm]; ok {
				uniq[pm] = append(uniq[pm], attrs.sources...)
			} else {
				uniq[pm] = attrs.sources
			}
		}
	}
	var pms = make([]DetectedPM, len(uniq))
	i := 0
	for pm, sources := range uniq {
		sort.StringSlice(sources).Sort()
		pms[i] = DetectedPM{Name: pm.Name, Category: pm.Category, Sources: sources}
		i++
	}

	slices.SortFunc(pms, func(a, b DetectedPM) int {
		return strings.Compare(a.Name, b.Name)
	})

	return pms
}

func (s *detectionStore) add(pm *PackageManager, src string, certain bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.byCategory == nil {
		s.byCategory = make(map[string]map[*PackageManager]detectedAttributes)
	}

	// Nothing is found in this category yet.
	inCategory, found := s.byCategory[pm.Category]
	if !found {
		s.byCategory[pm.Category] = make(map[*PackageManager]detectedAttributes)
		s.byCategory[pm.Category][pm] = detectedAttributes{
			sources: []string{src},
			certain: certain,
		}
		return nil
	}

	// The same package manager has been found.
	if existing, ok := inCategory[pm]; ok {
		existing.sources = append(existing.sources, src)
		if certain {
			existing.certain = certain
		}
		s.byCategory[pm.Category][pm] = existing
		return nil
	}

	for k, attr := range inCategory {
		if certain && k != pm {
			if attr.certain {
				return fmt.Errorf(
					"conflicting PMs found for source %s, category %s (%s vs %s)",
					src, pm.Category, pm, k)
			}
			delete(inCategory, k)
		} else if k == pm {
			attr.sources = append(attr.sources, src)
		}
	}

	return nil
}
