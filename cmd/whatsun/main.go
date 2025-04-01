package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/IGLOU-EU/go-wildcard/v2"
	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/upsun/whatsun"
	"github.com/upsun/whatsun/pkg/eval"
	"github.com/upsun/whatsun/pkg/rules"
)

var ignore = flag.String("ignore", "",
	"Comma-separated list of paths (or patterns) to ignore, adding to defaults")
var customRulesets = flag.String("rulesets", "",
	"Path to a custom ruleset directory (replacing the default embedded rulesets)")
var filter = flag.String("filter", "",
	"Filter the rulesets to ones matching the wildcard pattern(s), separated by commas")
var asJSON = flag.Bool("json", false, "Print output in JSON format")
var noMetadata = flag.Bool("no-meta", false, "Skip calculating and returning metadata.")
var simple = flag.Bool("simple", false,
	"Only output a simple list of results per path, with no metadata or other context.")

func main() {
	flag.Parse()
	path := "."
	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		fatalErr(err)
	}

	var analyzerConfig = &rules.AnalyzerConfig{
		IgnoreDirs:      strings.Split(*ignore, ","),
		DisableMetadata: *noMetadata || *simple,
	}
	var rulesets []rules.RulesetSpec
	if *customRulesets != "" {
		var err error
		dirName, fileName := filepath.Split(*customRulesets)
		if dirName == "" {
			dirName = "."
		}
		rulesets, err = rules.LoadFromYAMLDir(os.DirFS(dirName), fileName)
		if err != nil {
			fatalf("failed to load custom rulesets: %v", err)
		}
		if len(rulesets) == 0 {
			fatalf("no rulesets found in directory: %s", *customRulesets)
		}
		userCacheDir, err := os.UserCacheDir()
		if err != nil {
			fatalf("failed to get user cache dir: %v", err)
		}
		cacheDir := filepath.Join(userCacheDir, "whatsun")
		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			fatalf("failed to create cache dir: %v", err)
		}
		cache, err := eval.NewFileCache(filepath.Join(cacheDir, "expr.cache"))
		if err != nil {
			fatalf("failed to create file cache: %v", err)
		}
		defer cache.Save() //nolint:errcheck
		analyzerConfig.CELExpressionCache = cache
	} else {
		var err error
		rulesets, err = whatsun.LoadRulesets()
		if err != nil {
			fatalErr(err)
		}
		exprCache, err := whatsun.LoadExpressionCache()
		if err != nil {
			fatalErr(err)
		}
		analyzerConfig.CELExpressionCache = exprCache
	}

	if *filter != "" {
		patterns := strings.Split(*filter, ",")
		var filtered = make([]rules.RulesetSpec, 0, len(rulesets))
		for _, rs := range rulesets {
			for _, p := range patterns {
				if wildcard.Match(strings.TrimSpace(p), rs.GetName()) {
					filtered = append(filtered, rs)
					break
				}
			}
		}
		rulesets = filtered
	}

	if len(rulesets) == 0 {
		fatalf("No rulesets found")
	}

	analyzer, err := rules.NewAnalyzer(rulesets, analyzerConfig)
	if err != nil {
		fatalErr(err)
	}

	start := time.Now()

	reports, err := analyzer.Analyze(context.Background(), os.DirFS(absPath), ".")
	if err != nil {
		fatalf("analysis failed: %v", err)
	}

	if len(reports) == 0 {
		if *asJSON {
			fmt.Fprint(os.Stdout, "[]\n")
			return
		}
		fatalf("No results found")
		return
	}

	calcTime := time.Since(start)

	if *simple {
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
			fmt.Fprintf(os.Stdout, "%s\t%s\n", path, strings.Join(pathResults[path], ", "))
		}
		return
	}

	if *asJSON {
		if err := json.NewEncoder(os.Stdout).Encode(reports); err != nil {
			fatalErr(err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "Received result in %s:\n", calcTime)

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
		fmt.Fprintln(os.Stdout, "\nRuleset:", name)
		tbl := table.NewWriter()
		tbl.SetOutputMirror(os.Stdout)
		if *noMetadata {
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
			if *noMetadata {
				tbl.AppendRow(table.Row{report.Path, report.Result, strings.Join(report.Groups, ", ")})
			} else {
				tbl.AppendRow(table.Row{report.Path, report.Result, strings.Join(report.Groups, ", "), with})
			}
		}
		tbl.Render()
	}
}

func fatalErr(err error) {
	fatalf("%v", err)
}

func fatalf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func isEmpty(v any) bool {
	if v == nil || v == "" {
		return true
	}
	val := reflect.ValueOf(v)
	return val.IsZero() || val.Len() == 0
}
