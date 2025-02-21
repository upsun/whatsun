package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"

	"what/internal/rules"
)

var cpuprofile = flag.String("cpuprofile", "", "Write CPU profile to a file")
var ignore = flag.String("ignore", "", "Comma-separated list of paths (or patterns) to ignore, adding to defaults")

func main() {
	flag.Parse()
	if cpuprofile != nil && *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
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
	f, err := os.Open(absPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

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

	fmt.Fprintf(os.Stdout, "Received result in %s:\n", time.Since(start))

	names := make([]string, 0, len(results))
	for name := range results {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintln(os.Stdout, "\nRuleset:", name)
		tbl := table.NewWriter()
		tbl.SetOutputMirror(os.Stdout)
		tbl.AppendHeader(table.Row{"Path", "Result", "Groups", "With"})

		paths := make([]string, 0, len(results[name].Paths))
		for path := range results[name].Paths {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, p := range paths {
			for _, report := range results[name].Paths[p] {
				if !report.Sure {
					continue
				}
				var with string
				if len(report.With) > 0 {
					for k, v := range report.With {
						if v.Error == "" {
							with += fmt.Sprintf("%s: %s\n", k, v.Value)
						}
					}
					with = strings.TrimSpace(with)
				}
				tbl.AppendRow(table.Row{p, report.Result, strings.Join(report.Groups, ", "), with})
			}
		}
		tbl.Render()
	}
}
