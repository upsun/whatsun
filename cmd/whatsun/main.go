package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/IGLOU-EU/go-wildcard/v2"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/upsun/whatsun"
	"github.com/upsun/whatsun/pkg/eval"
	"github.com/upsun/whatsun/pkg/files"
	"github.com/upsun/whatsun/pkg/rules"
)

func main() {
	var cnf = &config{}
	var cmd = &cobra.Command{
		Use:   "whatsun [path]",
		Short: "Analyze a code repository",
		Args:  cobra.RangeArgs(0, 1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cnf.path = "."
			if len(args) > 0 {
				cnf.path = args[0]
			}

			return run(
				cmd.Context(),
				cnf,
				cmd.OutOrStdout(),
				cmd.ErrOrStderr(),
			)
		},
	}

	// Add flags to the command
	cmd.Flags().StringVar(&cnf.customRulesets, "rulesets", "",
		"Path to a custom ruleset directory (replacing the default embedded rulesets)")
	cmd.Flags().StringSliceVar(&cnf.ignore, "ignore", []string{},
		"Paths (or patterns) to ignore, adding to defaults")
	cmd.Flags().StringSliceVar(&cnf.filter, "filter", []string{},
		"Filter the rulesets to ones matching the wildcard pattern(s)")
	cmd.Flags().BoolVar(&cnf.asJSON, "json", false,
		"Print output in JSON format")
	cmd.Flags().BoolVar(&cnf.noMetadata, "no-meta", false,
		"Skip calculating and returning metadata.")
	cmd.Flags().BoolVar(&cnf.simple, "simple", false,
		"Only output a simple list of results per path, with no metadata or other context.")
	cmd.Flags().BoolVar(&cnf.tree, "tree", false,
		"Only output a file tree")

	if err := cmd.Execute(); err != nil {
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
	path           string
	ignore         []string
	customRulesets string
	filter         []string
	asJSON         bool
	noMetadata     bool
	simple         bool
	tree           bool
}

func run(ctx context.Context, cnf *config, stdout, stderr io.Writer) error {
	note := func(format string, args ...any) {
		fmt.Fprintf(stderr, strings.TrimRight(color.CyanString(format+"\n"), "\n"), args...)
	}

	var (
		fsys fs.FS
		err  error
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
		IgnoreDirs:      cnf.ignore,
		DisableMetadata: cnf.noMetadata || cnf.simple,
	}
	var rulesets []rules.RulesetSpec
	if cnf.customRulesets != "" {
		var err error
		dirName, fileName := filepath.Split(cnf.customRulesets)
		if dirName == "" {
			dirName = "."
		}
		rulesets, err = rules.LoadFromYAMLDir(os.DirFS(dirName), fileName)
		if err != nil {
			return fmt.Errorf("failed to load custom rulesets: %v", err)
		}
		if len(rulesets) == 0 {
			return fmt.Errorf("no rulesets found in directory: %s", cnf.customRulesets)
		}
		userCacheDir, err := os.UserCacheDir()
		if err != nil {
			return fmt.Errorf("failed to get user cache dir: %v", err)
		}
		cacheDir := filepath.Join(userCacheDir, "whatsun")
		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			return fmt.Errorf("failed to create cache dir: %v", err)
		}
		cache, err := eval.NewFileCache(filepath.Join(cacheDir, "expr.cache"))
		if err != nil {
			return fmt.Errorf("failed to create file cache: %v", err)
		}
		defer cache.Save() //nolint:errcheck
		analyzerConfig.CELExpressionCache = cache
	} else {
		var err error
		rulesets, err = whatsun.LoadRulesets()
		if err != nil {
			return err
		}
		exprCache, err := whatsun.LoadExpressionCache()
		if err != nil {
			return err
		}
		analyzerConfig.CELExpressionCache = exprCache
	}

	if len(cnf.filter) > 0 {
		var filtered = make([]rules.RulesetSpec, 0, len(rulesets))
		for _, rs := range rulesets {
			for _, p := range cnf.filter {
				if wildcard.Match(strings.TrimSpace(p), rs.GetName()) {
					filtered = append(filtered, rs)
					break
				}
			}
		}
		rulesets = filtered
	}

	if len(rulesets) == 0 {
		return fmt.Errorf("no rulesets found")
	}

	analyzer, err := rules.NewAnalyzer(rulesets, analyzerConfig)
	if err != nil {
		return err
	}

	start := time.Now()

	reports, err := analyzer.Analyze(ctx, fsys, ".")
	if err != nil {
		return fmt.Errorf("analysis failed: %v", err)
	}

	if len(reports) == 0 {
		if cnf.asJSON {
			fmt.Fprintln(stdout, "[]")
			return nil
		}
		return fmt.Errorf("no results found")
	}

	calcTime := time.Since(start)

	if cnf.simple {
		pathResults := map[string][]string{} // Map of path to results.
		for _, report := range reports {
			if report.Maybe {
				continue
			}
			pathResults[report.Path] = append(pathResults[report.Path], report.Result)
		}
		paths := make([]string, 0, len(pathResults))
		for path := range pathResults {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			fmt.Fprintf(stdout, "%s\t%s\n", path, strings.Join(pathResults[path], ", "))
		}
		return nil
	}

	if cnf.asJSON {
		if err := json.NewEncoder(stdout).Encode(reports); err != nil {
			return err
		}
		return nil
	}

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
		if cnf.noMetadata {
			tbl.AppendHeader(table.Row{"Path", "Result", "Groups"})
		} else {
			tbl.AppendHeader(table.Row{"Path", "Result", "Groups", "With"})
		}

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
			if cnf.noMetadata {
				tbl.AppendRow(table.Row{report.Path, report.Result, strings.Join(report.Groups, ", ")})
			} else {
				tbl.AppendRow(table.Row{report.Path, report.Result, strings.Join(report.Groups, ", "), with})
			}
		}
		tbl.Render()
	}

	return nil
}
