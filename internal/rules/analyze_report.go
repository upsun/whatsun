package rules

import (
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
		if len(res.Directories) == 0 {
			s += "\n[No results]\n"
			continue
		}
		s += "\nPath\tMatches\n"
		lines := make([]string, 0, len(res.Directories))
		for dir, matches := range res.Directories {
			lines = append(lines, fmt.Sprintf("%s\t%+v", dir, matches))
		}
		sort.Strings(lines)
		s += strings.Join(lines, "\n")
		s += "\n"
	}

	return strings.TrimRight(s, "\n")
}

type Result struct {
	Directories map[string][]Report
}

type Report struct {
	Result string
	Sure   bool
	Err    error
	Rules  []string
	With   map[string]string
}

func matchToReport(ev *eval.Evaluator, input any, rules map[string]Rule, match Match) Report {
	rep := Report{
		Result: match.Result,
		Sure:   match.Sure,
		Err:    match.Err,
		Rules:  make([]string, len(match.Rules)),
	}

	var reports []Report
	for i, ruleName := range match.Rules {
		rule, ok := rules[ruleName]
		if !ok {
			continue
		}
		rep.Rules[i] = rule.Name
		if len(rule.With) == 0 {
			continue
		}
		if rep.With == nil {
			rep.With = make(map[string]string)
		}
		for name, expr := range rule.With {
			val, err := ev.Eval(expr, input)
			if err != nil {
				rep.With[name] = fmt.Sprint("[ERROR] ", err.Error())
				continue
			}
			rep.With[name] = fmt.Sprint(val)
		}
		reports = append(reports, rep)
	}
	slices.Sort(rep.Rules)

	return rep
}
