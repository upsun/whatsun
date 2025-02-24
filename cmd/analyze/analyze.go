package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime/pprof"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"

	"what/internal/rules"
)

var cpuprofile = flag.String("cpuprofile", "", "Write CPU profile to a file")
var heapprofile = flag.String("heapprofile", "", "Write heap profile to a file")
var allocsprofile = flag.String("allocsprofile", "", "Write allocations profile to a file")
var ignore = flag.String("ignore", "", "Comma-separated list of paths (or patterns) to ignore, adding to defaults")

func main() {
	flag.Parse()
	if cpuprofile != nil && *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	path := "."
	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	ignoreSlice := strings.Split(*ignore, ",")

	analyzer, err := rules.NewAnalyzer(ignoreSlice)
	if err != nil {
		log.Fatal(err)
	}
	start := time.Now()

	results, err := analyzer.Analyze(context.TODO(), os.DirFS(absPath), ".")
	if err != nil {
		log.Fatal(err)
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
			if !report.Sure {
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

	if heapprofile != nil && *heapprofile != "" {
		f, err := os.Create(*heapprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal(err)
		}
	}
	if allocsprofile != nil && *allocsprofile != "" {
		f, err := os.Create(*allocsprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
			log.Fatal(err)
		}
	}
}

func isEmpty(v any) bool {
	if v == nil || v == "" {
		return true
	}
	val := reflect.ValueOf(v)
	return val.IsZero() || val.Len() == 0
}
