package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

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

	result, err := analyzer.Analyze(context.TODO(), os.DirFS(absPath), ".")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(os.Stdout, "Received result in %s:\n", time.Since(start))
	fmt.Fprintln(os.Stdout, result)
}
