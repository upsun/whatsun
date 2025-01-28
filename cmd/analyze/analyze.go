package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/sync/errgroup"

	"what"
	"what/analysis"
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

	resultChan := make(chan resultContext)
	err = analyze(context.TODO(), os.DirFS(path), ".", resultChan)
	if err != nil {
		log.Fatal(err)
	}

	for c := range resultChan {
		log.Printf(`received result from analyzer "%s"`, c.Analyzer.Name())
		for _, r := range c.results {
			fmt.Printf("payload: %s, reason: %s\n", r.Payload, r.Reason)
		}
	}
}

type resultContext struct {
	what.Analyzer
	results []what.Result
}

func analyze(ctx context.Context, fsys fs.FS, root string, resultChan chan<- resultContext) error {
	analyzers := []what.Analyzer{
		&analysis.Apps{MaxDepth: 3},
	}

	var err error
	go func() {
		eg := errgroup.Group{}
		eg.SetLimit(runtime.GOMAXPROCS(0))
		defer close(resultChan)
		for _, a := range analyzers {
			a := a
			eg.Go(func() error {
				result, err := a.Analyze(ctx, fsys, root)
				if err != nil {
					if errors.Is(err, what.ErrNotApplicable) {
						return nil
					}
					return err
				}
				resultChan <- resultContext{
					Analyzer: a,
					results:  result,
				}
				return nil
			})
		}
		err = eg.Wait()
	}()

	return err
}
