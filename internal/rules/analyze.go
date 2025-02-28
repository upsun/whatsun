// Package rules applies rules and combines the results.
package rules

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/google/cel-go/common/types"
	"golang.org/x/sync/errgroup"

	"what/internal/eval"
	"what/internal/eval/celfuncs"
	"what/internal/fsgitignore"
	"what/internal/searchfs"
)

type Analyzer struct {
	rulesets  []RulesetSpec
	evaluator *eval.Evaluator
	ignore    []string
}

func NewAnalyzer(rulesets []RulesetSpec, ev *eval.Evaluator, ignore []string) *Analyzer {
	return &Analyzer{evaluator: ev, rulesets: rulesets, ignore: ignore}
}

func (a *Analyzer) Analyze(ctx context.Context, fsys fs.FS, root string) (RulesetReports, error) {
	fsys = searchfs.New(fsys)

	var (
		numWorkers = runtime.GOMAXPROCS(0)
		dirChan    = make(chan string, numWorkers)
		errGroup   errgroup.Group
	)
	errGroup.Go(func() error {
		defer close(dirChan)
		return a.collectDirectories(ctx, fsys, root, dirChan)
	})

	type reportsKeyed struct {
		set     string
		reports []Report
	}

	var reportsChan = make(chan reportsKeyed, numWorkers)
	errGroup.Go(func() error {
		var dirGroup errgroup.Group
		dirGroup.SetLimit(numWorkers)
		defer close(reportsChan)
		for path := range dirChan {
			path := path
			dirGroup.Go(func() error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default: // Continue only if the context was not canceled.
				}
				for _, ruleset := range a.rulesets {
					subReports, err := a.applyRuleset(ruleset, fsys, path)
					if err != nil {
						return err
					}
					reportsChan <- reportsKeyed{ruleset.GetName(), subReports}
				}
				return nil
			})
		}
		return dirGroup.Wait()
	})

	var rulesetReports = make(RulesetReports)
	errGroup.Go(func() error {
		var mux sync.Mutex
		for rk := range reportsChan {
			mux.Lock()
			rulesetReports[rk.set] = append(rulesetReports[rk.set], rk.reports...)
			mux.Unlock()
		}
		return nil
	})

	if err := errGroup.Wait(); err != nil {
		return nil, err
	}

	for k, rr := range rulesetReports {
		slices.SortFunc(rr, func(a, b Report) int {
			return strings.Compare(a.Path, b.Path)
		})
		rulesetReports[k] = rr
	}

	return rulesetReports, nil
}

func (a *Analyzer) collectDirectories(ctx context.Context, fsys fs.FS, root string, dirChan chan<- string) error {
	var ignorePatterns = defaultIgnorePatterns
	if len(a.ignore) > 0 {
		ignorePatterns = append(ignorePatterns, fsgitignore.ParsePatterns(a.ignore, fsgitignore.Split(root))...)
	}
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default: // Continue only if the context was not canceled.
		}
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		// Hard-limit the directory depth to 16.
		if strings.Count(path, string(os.PathSeparator)) >= 16 {
			return filepath.SkipDir
		}
		if d.Name() == ".git" || d.Name() == "node_modules" {
			return fs.SkipDir
		}
		if gitignore.NewMatcher(ignorePatterns).Match(fsgitignore.Split(path), true) {
			return fs.SkipDir
		}
		patterns, err := fsgitignore.ParseIgnoreFiles(fsys, path)
		if err != nil {
			return err
		}
		ignorePatterns = append(ignorePatterns, patterns...)
		dirChan <- path
		return nil
	})
}

func (a *Analyzer) evalFuncForDirectory(dir string, celInput map[string]any, gitIgnores *ignoreStore) func(rule RuleSpec) (bool, error) {
	dirSplit := fsgitignore.Split(dir)

	return func(rule RuleSpec) (bool, error) {
		if ri, ok := rule.(ignorer); ok {
			if m := gitIgnores.getMatcher(ri); m != nil && m.Match(dirSplit, true) {
				return false, nil
			}
		}

		val, err := a.evaluator.Eval(rule.GetCondition(), celInput)
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

func (a *Analyzer) applyRuleset(rs RulesetSpec, fsys fs.FS, path string) ([]Report, error) {
	var ignores = &ignoreStore{}
	var celInput = celfuncs.FilesystemInput(fsys, path)

	matches, err := FindMatches(rs.GetRules(), a.evalFuncForDirectory(path, celInput, ignores))
	if err != nil {
		return nil, fmt.Errorf("in directory %s: %w", path, err)
	}

	var reports = make([]Report, len(matches))
	for i, m := range matches {
		reports[i] = matchToReport(a.evaluator, celInput, m, path)
	}

	return reports, nil
}
