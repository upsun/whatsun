package rules

import "sort"

// Report contains results and other metadata after applying rules.
type Report struct {
	Ruleset string `json:"ruleset,omitempty"`
	Path    string `json:"path,omitempty"`

	Result string   `json:"result,omitempty"`
	Error  string   `json:"error,omitempty"`
	Rules  []string `json:"rules,omitempty"`

	Maybe bool `json:"maybe,omitempty"`

	Groups []string               `json:"groups,omitempty"`
	With   map[string]ReportValue `json:"with,omitempty"`
}

// ReportValue contains a reported value or an error message.
type ReportValue struct {
	Value any    `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

func sortedMapKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}

	var s = make([]string, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Strings(s)
	return s
}
