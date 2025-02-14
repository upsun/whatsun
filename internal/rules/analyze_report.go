package rules

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"what/internal/eval"
)

type Results map[string]Result

func (r Results) String() string {
	if r == nil {
		return "[no results]"
	}

	names := make([]string, 0, len(r))
	for name := range r {
		names = append(names, name)
	}
	sort.Strings(names)

	s := ""
	for _, name := range names {
		s += fmt.Sprintf("\nRuleset: %s", name)
		res := r[name]
		if len(res.Paths) == 0 {
			s += "\n[No results]\n"
			continue
		}
		s += "\nPath\tMatches\n"
		lines := make([]string, 0, len(res.Paths))
		for dir, matches := range res.Paths {
			b, _ := json.Marshal(matches)
			lines = append(lines, dir+"\t"+string(b))
		}
		sort.Strings(lines)
		s += strings.Join(lines, "\n")
		s += "\n"
	}

	return strings.TrimRight(s, "\n")
}

type Result struct {
	Paths map[string][]Report `json:"directories"`
}

type Report struct {
	Result string              `json:"result,omitempty"`
	Sure   bool                `json:"sure,omitempty"`
	Error  string              `json:"error,omitempty"`
	Rules  []string            `json:"rules,omitempty"`
	Groups []string            `json:"groups,omitempty"`
	With   map[string]Metadata `json:"with,omitempty"`
}

type Metadata struct {
	Value any    `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

func matchToReport(ev *eval.Evaluator, input any, rules map[string]Rule, match Match) Report {
	rep := Report{
		Result: match.Result,
		Sure:   match.Sure,
		Rules:  make([]string, len(match.Rules)),
	}
	if match.Err != nil {
		rep.Error = match.Err.Error()
	}

	var groupMap = make(map[string]struct{})
	var reports []Report
	for i, ruleName := range match.Rules {
		rule, ok := rules[ruleName]
		if !ok {
			continue
		}
		if rule.Group != "" {
			groupMap[rule.Group] = struct{}{}
		}
		rep.Rules[i] = rule.Name
		if len(rule.With) == 0 {
			continue
		}
		if rep.With == nil {
			rep.With = make(map[string]Metadata)
		}
		for name, expr := range rule.With {
			val, err := ev.Eval(expr, input)
			if err != nil {
				rep.With[name] = Metadata{Error: err.Error()}
				continue
			}
			rep.With[name] = Metadata{Value: val.Value()}
		}
		reports = append(reports, rep)
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
