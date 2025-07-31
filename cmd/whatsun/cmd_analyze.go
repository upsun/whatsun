package main

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/upsun/whatsun"
	"github.com/upsun/whatsun/pkg/rules"
)

func analyzeCmd() *cobra.Command {
	var ignore []string
	cmd := &cobra.Command{
		Use:   "analyze [path]",
		Short: "Analyze a code repository and show results",
		Args:  cobra.RangeArgs(0, 1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return runAnalyze(cmd.Context(), path, ignore, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().StringSliceVar(&ignore, "ignore", []string{},
		"Paths (or patterns) to ignore, adding to defaults.")

	return cmd
}

func runAnalyze(ctx context.Context, path string, ignore []string, stdout, stderr io.Writer) error {
	fsys, disableGitIgnore, err := setupFileSystem(ctx, path, stderr)
	if err != nil {
		return err
	}

	rulesets, err := whatsun.LoadRulesets()
	if err != nil {
		return err
	}

	exprCache, err := whatsun.LoadExpressionCache()
	if err != nil {
		return err
	}

	analyzerConfig := &rules.AnalyzerConfig{
		IgnoreDirs:         ignore,
		DisableGitIgnore:   disableGitIgnore,
		CELExpressionCache: exprCache,
	}

	start := time.Now()

	analyzer, err := rules.NewAnalyzer(rulesets, analyzerConfig)
	if err != nil {
		return err
	}

	reports, err := analyzer.Analyze(ctx, fsys, ".")
	if err != nil {
		return fmt.Errorf("analysis failed: %v", err)
	}

	calcTime := time.Since(start)
	return printAnalysisResults(reports, calcTime, stdout, stderr)
}

func printAnalysisResults(reports []rules.Report, calcTime time.Duration, stdout, stderr io.Writer) error {
	if len(reports) == 0 {
		return fmt.Errorf("no results found")
	}

	note := noter(stderr)
	note("Received result in %s:", calcTime)

	var byRuleset = make(map[string][]rules.Report)
	for _, report := range reports {
		byRuleset[report.Ruleset] = append(byRuleset[report.Ruleset], report)
	}

	names := make([]string, 0, len(byRuleset))
	for name := range byRuleset {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintln(stdout, "\nRuleset:", name)
		tbl := table.NewWriter()
		tbl.SetOutputMirror(stdout)
		tbl.AppendHeader(table.Row{"Path", "Result", "Groups", "With"})

		for _, report := range byRuleset[name] {
			if report.Maybe {
				continue
			}
			var with string
			if len(report.With) > 0 {
				for k, v := range report.With {
					if v.Error == "" && !isEmpty(v.Value) {
						with += fmt.Sprintf("%s: %s\n", k, v.Value)
					}
				}
				with = strings.TrimSpace(with)
			}
			tbl.AppendRow(table.Row{report.Path, report.Result, strings.Join(report.Groups, ", "), with})
		}
		tbl.Render()
	}

	return nil
}

func isEmpty(v any) bool {
	if v == nil || v == "" {
		return true
	}
	val := reflect.ValueOf(v)
	return val.IsZero() || val.Len() == 0
}
