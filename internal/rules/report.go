package rules

import (
	"slices"
	"sort"

	"what/internal/eval"
)

// RulesetReports collects reports for each ruleset (keyed by the ruleset name).
type RulesetReports map[string][]Report

// Report contains results and other metadata after applying rules.
type Report struct {
	Path   string   `json:"path"`
	Result string   `json:"result,omitempty"`
	Error  string   `json:"error,omitempty"`
	Rules  []string `json:"rules,omitempty"`

	Sure bool `json:"sure,omitempty"`

	Groups []string               `json:"groups,omitempty"`
	With   map[string]ReportValue `json:"with,omitempty"`
}

// ReportValue contains a reported value or an error message.
type ReportValue struct {
	Value any    `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

func matchToReport(ev *eval.Evaluator, input any, match Match, path string) Report {
	rep := Report{
		Path:   path,
		Result: match.Result,
		Sure:   match.Sure,
		Rules:  make([]string, len(match.Rules)),
	}
	if match.Err != nil {
		rep.Error = match.Err.Error()
	}

	var groupMap = make(map[string]struct{})
	for i, rule := range match.Rules {
		if rg, ok := rule.(WithGroups); ok {
			for _, g := range rg.GetGroups() {
				groupMap[g] = struct{}{}
			}
		}
		rep.Rules[i] = rule.GetName()
		if rm, ok := rule.(WithMetadata); ok && len(rm.GetMetadata()) > 0 {
			if rep.With == nil {
				rep.With = make(map[string]ReportValue)
			}
			for name, expr := range rm.GetMetadata() {
				val, err := ev.Eval(expr, input)
				if err != nil {
					rep.With[name] = ReportValue{Error: err.Error()}
					continue
				}
				rep.With[name] = ReportValue{Value: val.Value()}
			}
		}
	}
	rep.Groups = sortedMapKeys(groupMap)
	slices.Sort(rep.Rules)

	return rep
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
