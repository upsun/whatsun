package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/upsun/whatsun/pkg/rules"
)

// outputDepsPlain outputs dependencies in plain tab-separated format
func outputDepsPlain(deps []dependencyInfo, stdout io.Writer) {
	fmt.Fprintln(stdout, "Path\tTool\tName\tConstraint\tVersion")
	for _, depInfo := range deps {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n",
			depInfo.Path,
			depInfo.Dependency.ToolName,
			depInfo.Dependency.Name,
			depInfo.Dependency.Constraint,
			depInfo.Dependency.Version,
		)
	}
}

// outputAnalyzePlain outputs analysis reports in plain tab-separated format
func outputAnalyzePlain(reports []rules.Report, stdout io.Writer) {
	fmt.Fprintln(stdout, "Path\tRuleset\tResult\tGroups\tWith")
	for _, report := range reports {
		if report.Maybe {
			continue
		}
		var with string
		if len(report.With) > 0 {
			var parts []string
			for k, v := range report.With {
				if v.Error == "" && !isEmpty(v.Value) {
					parts = append(parts, fmt.Sprintf("%s: %s", k, v.Value))
				}
			}
			with = strings.Join(parts, "; ")
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n",
			report.Path,
			report.Ruleset,
			report.Result,
			strings.Join(report.Groups, ", "),
			with,
		)
	}
}
