package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/upsun/whatsun"
	"github.com/upsun/whatsun/pkg/files"
	"github.com/upsun/whatsun/pkg/rules"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "whatsun",
		Short: "Analyze a code repository",
	}

	// Main analyze command (default behavior)
	var analyzeCnf = &config{}
	var analyzeCmd = &cobra.Command{
		Use:   "analyze [path]",
		Short: "Analyze a code repository and show results",
		Args:  cobra.RangeArgs(0, 1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			analyzeCnf.path = "."
			if len(args) > 0 {
				analyzeCnf.path = args[0]
			}
			return run(
				cmd.Context(),
				analyzeCnf,
				cmd.OutOrStdout(),
				cmd.ErrOrStderr(),
			)
		},
	}
	analyzeCmd.Flags().StringSliceVar(&analyzeCnf.ignore, "ignore", []string{},
		"Paths (or patterns) to ignore, adding to defaults.")

	// Digest command
	var digestCnf = &config{digest: true}
	var digestCmd = &cobra.Command{
		Use:   "digest [path]",
		Short: "Output a digest of the repository including the file tree, reports, and the contents of selected files",
		Args:  cobra.RangeArgs(0, 1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			digestCnf.path = "."
			if len(args) > 0 {
				digestCnf.path = args[0]
			}
			return run(
				cmd.Context(),
				digestCnf,
				cmd.OutOrStdout(),
				cmd.ErrOrStderr(),
			)
		},
	}
	digestCmd.Flags().StringSliceVar(&digestCnf.ignore, "ignore", []string{},
		"Paths (or patterns) to ignore, adding to defaults.")

	// Tree command
	var treeCnf = &config{tree: true}
	var treeCmd = &cobra.Command{
		Use:   "tree [path]",
		Short: "Only output a file tree",
		Args:  cobra.RangeArgs(0, 1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			treeCnf.path = "."
			if len(args) > 0 {
				treeCnf.path = args[0]
			}
			return run(
				cmd.Context(),
				treeCnf,
				cmd.OutOrStdout(),
				cmd.ErrOrStderr(),
			)
		},
	}
	treeCmd.Flags().StringSliceVar(&treeCnf.ignore, "ignore", []string{},
		"Paths (or patterns) to ignore, adding to defaults.")

	// Add subcommands
	rootCmd.AddCommand(analyzeCmd, digestCmd, treeCmd)

	// Set analyze as the default command when no subcommand is provided
	rootCmd.RunE = analyzeCmd.RunE
	rootCmd.Args = analyzeCmd.Args
	rootCmd.ValidArgsFunction = analyzeCmd.ValidArgsFunction
	rootCmd.Flags().StringSliceVar(&analyzeCnf.ignore, "ignore", []string{},
		"Paths (or patterns) to ignore, adding to defaults.")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(color.RedString(err.Error()))
		os.Exit(1)
	}
}

func isEmpty(v any) bool {
	if v == nil || v == "" {
		return true
	}
	val := reflect.ValueOf(v)
	return val.IsZero() || val.Len() == 0
}

type config struct {
	path   string
	ignore []string
	digest bool
	tree   bool
}

func run(ctx context.Context, cnf *config, stdout, stderr io.Writer) error {
	note := func(format string, args ...any) {
		fmt.Fprintf(stderr, strings.TrimRight(color.CyanString(format+"\n"), "\n"), args...)
	}

	var (
		fsys             fs.FS
		err              error
		disableGitIgnore bool
	)
	if files.IsLocal(cnf.path) {
		note("Processing local path: %s", cnf.path)
		fsys, err = files.LocalFS(cnf.path)
		if err != nil {
			return err
		}
	} else {
		preClone := time.Now()
		note("Cloning remote repository (into memory): %s", cnf.path)
		fsys, err = files.Clone(ctx, cnf.path, "")
		if err != nil {
			return err
		}
		note("Cloned repository in %v", time.Since(preClone).Truncate(time.Millisecond))
		disableGitIgnore = true
	}

	if cnf.tree {
		result, err := files.GetTree(fsys, files.MinimalTreeConfig)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, strings.Join(result, "\n"))
		return nil
	}

	var analyzerConfig = &rules.AnalyzerConfig{
		IgnoreDirs:       cnf.ignore,
		DisableGitIgnore: disableGitIgnore,
	}
	rulesets, err := whatsun.LoadRulesets()
	if err != nil {
		return err
	}
	exprCache, err := whatsun.LoadExpressionCache()
	if err != nil {
		return err
	}
	analyzerConfig.CELExpressionCache = exprCache

	if len(rulesets) == 0 {
		return fmt.Errorf("no rulesets found")
	}

	if cnf.digest {
		digestCnf, err := files.DefaultDigestConfig()
		if err != nil {
			return err
		}
		digestCnf.DisableGitIgnore = disableGitIgnore
		digestCnf.IgnoreFiles = cnf.ignore
		digestCnf.Rulesets = rulesets
		digester, err := files.NewDigester(fsys, digestCnf)
		if err != nil {
			return err
		}
		digest, err := digester.GetDigest(ctx)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(digest)
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

	if len(reports) == 0 {
		return fmt.Errorf("no results found")
	}

	calcTime := time.Since(start)

	fmt.Fprintf(stderr, "Received result in %s:\n", calcTime)

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
