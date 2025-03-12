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

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"golang.org/x/sync/errgroup"

	"github.com/upsun/whatsun/internal/fsgitignore"
	"github.com/upsun/whatsun/internal/searchfs"
	"github.com/upsun/whatsun/pkg/eval"
	"github.com/upsun/whatsun/pkg/eval/celfuncs"
)

type AnalyzerConfig struct {
	CELEnvOptions      []cel.EnvOption // Optional custom CEL environment options, replacing the default.
	CELExpressionCache eval.Cache      // Optional expression cache: ideally it should cover the expected expressions.

	// DisableGitIgnore disables handling of .gitignore and .git/info/exclude files.
	//
	// The IgnoreDirs setting will still be respected, and certain directories will
	// always be ignored (namely .git and node_modules). Rules that implement the
	// Ignorer interface will also still be respected.
	DisableGitIgnore bool

	IgnoreDirs []string // Additional directory ignore rules, using git's exclude syntax.
}

type Analyzer struct {
	evaluator *eval.Evaluator
	rulesets  []RulesetSpec
	cnf       *AnalyzerConfig
}

func NewAnalyzer(rulesets []RulesetSpec, cnf *AnalyzerConfig) (*Analyzer, error) {
	if cnf == nil {
		cnf = &AnalyzerConfig{}
	}
	if cnf.CELEnvOptions == nil {
		cnf.CELEnvOptions = celfuncs.DefaultEnvOptions()
	}
	ev, err := eval.NewEvaluator(&eval.Config{
		EnvOptions: cnf.CELEnvOptions,
		Cache:      cnf.CELExpressionCache,
	})
	if err != nil {
		return nil, err
	}

	return &Analyzer{evaluator: ev, rulesets: rulesets, cnf: cnf}, nil
}

func (a *Analyzer) Analyze(ctx context.Context, fsys fs.FS, root string) (RulesetReports, error) {
	fsys = searchfs.New(fsys)

	var (
		// Limit the number of per-directory workers to 2 less than GOMAXPROCS.
		numWorkers = max(1, runtime.GOMAXPROCS(0)-2)
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
		for rk := range reportsChan {
			rulesetReports[rk.set] = append(rulesetReports[rk.set], rk.reports...)
		}
		return nil
	})

	if err := errGroup.Wait(); err != nil {
		return nil, err
	}

	for _, rr := range rulesetReports {
		slices.SortFunc(rr, func(a, b Report) int {
			return strings.Compare(a.Path, b.Path)
		})
	}

	return rulesetReports, nil
}

func (a *Analyzer) collectDirectories(ctx context.Context, fsys fs.FS, root string, dirChan chan<- string) error {
	var ignorePatterns = defaultIgnorePatterns
	if len(a.cnf.IgnoreDirs) > 0 {
		ignorePatterns = append(ignorePatterns, fsgitignore.ParsePatterns(a.cnf.IgnoreDirs, fsgitignore.Split(root))...)
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
		if !a.cnf.DisableGitIgnore {
			patterns, err := fsgitignore.ParseIgnoreFiles(fsys, path)
			if err != nil {
				return err
			}
			ignorePatterns = append(ignorePatterns, patterns...)
		}
		dirChan <- path
		return nil
	})
}

func (a *Analyzer) evalFuncForDirectory(dir string, celInput map[string]any) func(rule RuleSpec) (bool, error) {
	dirSplit := fsgitignore.Split(dir)

	return func(rule RuleSpec) (bool, error) {
		if ri, ok := rule.(Ignorer); ok {
			if m := getIgnoreMatcher(ri); m != nil && m.Match(dirSplit, true) {
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

		return bool(asBool.(types.Bool)), nil //nolint:errcheck // the type is known
	}
}

func (a *Analyzer) applyRuleset(rs RulesetSpec, fsys fs.FS, path string) ([]Report, error) {
	var celInput = celfuncs.FilesystemInput(fsys, path)

	matches, err := FindMatches(rs.GetRules(), a.evalFuncForDirectory(path, celInput))
	if err != nil {
		return nil, fmt.Errorf("in directory %s: %w", path, err)
	}

	var reports = make([]Report, len(matches))
	for i, m := range matches {
		reports[i] = matchToReport(a.evaluator, celInput, m, path)
	}

	return reports, nil
}
