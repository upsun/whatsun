package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"what/analyzers/apps"

	"golang.org/x/sync/errgroup"

	"what"
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
	analyze(context.TODO(), []what.Analyzer{&apps.Analyzer{}}, os.DirFS(absPath), resultChan)

	errOut := bufio.NewWriter(os.Stderr)
	defer errOut.Flush()

	for r := range resultChan {
		if r.err != nil {
			log.Fatal(r.err)
		}
		fmt.Fprintf(errOut, "Received result from analyzer \"%s\":\n", r.Analyzer.GetName())
		fmt.Fprintln(errOut, r.Result.GetSummary())
		errOut.Flush()
	}
}

type resultContext struct {
	err error
	what.Analyzer
	what.Result
}

// analyze runs a list of analyzers and sends results.
func analyze(ctx context.Context, analyzers []what.Analyzer, fsys fs.FS, resultChan chan<- resultContext) {
	go func() {
		eg := errgroup.Group{}
		eg.SetLimit(runtime.GOMAXPROCS(0))
		defer close(resultChan)
		for _, a := range analyzers {
			a := a
			eg.Go(func() error {
				result, err := a.Analyze(ctx, fsys)
				if err != nil {
					return err
				}
				resultChan <- resultContext{
					Analyzer: a,
					Result:   result,
				}
				return nil
			})
		}
		err := eg.Wait()
		if err != nil {
			resultChan <- resultContext{err: err}
		}
	}()
}
