// Package rules applies rules and combines the results.
package rules

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/cel-go/common/types"
	"golang.org/x/sync/errgroup"

	"what/internal/eval"
	"what/internal/eval/celfuncs"
)

type Analyzer struct {
	config    map[string]Ruleset
	evaluator *eval.Evaluator
}

func NewAnalyzer() (*Analyzer, error) {
	cache, err := eval.NewFileCacheWithContent(exprCache, "")
	if err != nil {
		return nil, err
	}
	ev, err := eval.NewEvaluator(&eval.Config{Cache: cache, EnvOptions: DefaultEnvOptions()})
	if err != nil {
		return nil, err
	}
	return &Analyzer{evaluator: ev, config: Config}, nil
}

func (a *Analyzer) Analyze(_ context.Context, fsys fs.FS, root string) (Results, error) {
	var results = make(Results, len(a.config))
	for name, rs := range a.config {
		res, err := a.applyRuleset(&rs, fsys, root)
		if err != nil {
			return nil, err
		}
		results[name] = Result{Paths: res}
	}

	return results, nil
}

func (a *Analyzer) evalWithInput(input any) func(string) (bool, error) {
	return func(condition string) (bool, error) {
		val, err := a.evaluator.Eval(condition, input)
		if err != nil {
			return false, err
		}

		asBool := val.ConvertToType(types.BoolType)
		if types.IsError(asBool) {
			return false, fmt.Errorf("%v", asBool)
		}

		return bool(asBool.(types.Bool)), nil
	}
}

type dirReport struct {
	Path   string
	Report Report
}

func (a *Analyzer) applyRuleset(rs *Ruleset, fsys fs.FS, root string) (map[string][]Report, error) {
	matcher := &Matcher{rs.Rules}

	var directoryPaths []string
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		// Calculate depth.
		// In a structure ./a/b/c then files in the root are level 0, in "a" are in level 1, in "b" are level 2, etc.
		var depth int
		if path != "." {
			depth = strings.Count(path, string(os.PathSeparator)) + 1
		}

		n := d.Name()
		// TODO implement actual gitignore
		if len(n) > 1 && n[0] == '.' {
			return filepath.SkipDir
		}
		switch n {
		case "vendor", "node_modules", "packages", "pkg", "tests", "logs", "doc", "docs", "bin", "dist",
			"__pycache__", "venv", "virtualenv", "target", "out", "build", "obj", "elm-stuff":
			return filepath.SkipDir
		}

		if d.IsDir() {
			directoryPaths = append(directoryPaths, path)
		}

		if depth >= min(rs.MaxDepth, 10) {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var dirReports []dirReport

	eg := errgroup.Group{}
	eg.SetLimit(runtime.GOMAXPROCS(0))
	for _, d := range directoryPaths {
		eg.Go(func() error {
			input := celfuncs.FilesystemInput(fsys, d)
			matches, err := matcher.Match(a.evalWithInput(input))
			if err != nil {
				return fmt.Errorf("in directory %s: %w", d, err)
			}
			for _, m := range matches {
				dirReports = append(dirReports, struct {
					Path   string
					Report Report
				}{Path: d, Report: matchToReport(a.evaluator, input, rs.Rules, m)})
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	var reports = make(map[string][]Report, len(directoryPaths))
	for _, r := range dirReports {
		reports[r.Path] = append(reports[r.Path], r.Report)
	}

	return reports, err
}
