package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"

	"what"
	"what/pkg/eval"
	"what/pkg/rules"
)

var ignore = flag.String("ignore", "", "Comma-separated list of paths (or patterns) to ignore, adding to defaults")
var customRulesets = flag.String("rulesets", "", "Path to a custom ruleset directory (replacing the default embedded rulesets)")

func main() {
	flag.Parse()
	path := "."
	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	var analyzerConfig = &rules.AnalyzerConfig{
		IgnoreDirs: strings.Split(*ignore, ","),
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
			log.Fatalf("failed to load custom rulesets: %v", err)
		}
		if len(rulesets) == 0 {
			log.Fatalf("no rulesets found in directory: %s", *customRulesets)
		}
		userCacheDir, err := os.UserCacheDir()
		if err != nil {
			log.Fatalf("failed to get user cache dir: %v", err)
		}
		cacheDir := filepath.Join(userCacheDir, "what")
		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			log.Fatalf("failed to create cache dir: %v", err)
		}
		cache, err := eval.NewFileCache(filepath.Join(cacheDir, "expr.cache"))
		if err != nil {
			log.Fatalf("failed to create file cache: %v", err)
		}
		defer cache.Save()
		analyzerConfig.CELExpressionCache = cache
	} else {
		var err error
		rulesets, err = what.LoadRulesets()
		if err != nil {
			log.Fatal(err)
		}
		exprCache, err := what.LoadExpressionCache()
		if err != nil {
			log.Fatal(err)
		}
		analyzerConfig.CELExpressionCache = exprCache
	}

	analyzer, err := rules.NewAnalyzer(rulesets, analyzerConfig)
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()

	results, err := analyzer.Analyze(context.Background(), os.DirFS(absPath), ".")
	if err != nil {
		log.Fatalf("analysis failed: %v", err)
	}

	names := make([]string, 0, len(results))
	for name := range results {
		if len(results[name]) == 0 {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	if len(names) == 0 {
		fmt.Fprintln(os.Stdout, "No results found")
		return
	}

	fmt.Fprintf(os.Stdout, "Received result in %s:\n", time.Since(start))

	for _, name := range names {
		fmt.Fprintln(os.Stdout, "\nRuleset:", name)
		tbl := table.NewWriter()
		tbl.SetOutputMirror(os.Stdout)
		tbl.AppendHeader(table.Row{"Path", "Result", "Groups", "With"})

		for _, report := range results[name] {
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
}

func isEmpty(v any) bool {
	if v == nil || v == "" {
		return true
	}
	val := reflect.ValueOf(v)
	return val.IsZero() || val.Len() == 0
}
