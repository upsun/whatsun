package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"what/internal/rules"
)

func main() {
	path := "."
	if len(os.Args) > 1 {
		path = os.Args[1]
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

	analyzer, err := rules.NewAnalyzer()
	if err != nil {
		log.Fatal(err)
	}
	start := time.Now()

	result, err := analyzer.Analyze(context.TODO(), os.DirFS(absPath))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(os.Stdout, "Received result in %s:\n", time.Since(start))
	fmt.Fprintln(os.Stdout, result)
}
