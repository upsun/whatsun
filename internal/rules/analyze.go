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

	dirs, err := a.collectDirectories(ctx, fsys, root)
	if err != nil {
		return nil, err
	}

	var rulesetReports = make(RulesetReports, len(a.rulesets))
	for _, ruleset := range a.rulesets {
		reports, err := a.applyRuleset(ruleset, fsys, dirs)
		if err != nil {
			return nil, err
		}
		rulesetReports[ruleset.GetName()] = reports
	}

	return rulesetReports, nil
}

func (a *Analyzer) collectDirectories(_ context.Context, fsys fs.FS, root string) ([]string, error) {
	var ignorePatterns = defaultIgnorePatterns
	if len(a.ignore) > 0 {
		ignorePatterns = append(ignorePatterns, fsgitignore.ParsePatterns(a.ignore, fsgitignore.Split(root))...)
	}
	var directoryPaths []string
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
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
		directoryPaths = append(directoryPaths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return directoryPaths, nil
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

func (a *Analyzer) applyRuleset(rs RulesetSpec, fsys fs.FS, directoryPaths []string) ([]Report, error) {
	var (
		rules   = rs.GetRules()
		ignores = &ignoreStore{}
		reports []Report
		mux     sync.Mutex
		eg      errgroup.Group
	)
	eg.SetLimit(runtime.GOMAXPROCS(0))
	for _, d := range directoryPaths {
		d := d // Copy the range variable, avoiding reuse in the goroutine.
		eg.Go(func() error {
			celInput := celfuncs.FilesystemInput(fsys, d)
			matches, err := FindMatches(rules, a.evalFuncForDirectory(d, celInput, ignores))
			if err != nil {
				return fmt.Errorf("in directory %s: %w", d, err)
			}
			var subReports = make([]Report, len(matches))
			for i, m := range matches {
				subReports[i] = matchToReport(a.evaluator, celInput, m, d)
			}
			mux.Lock()
			reports = append(reports, subReports...)
			mux.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	slices.SortFunc(reports, func(a, b Report) int {
		return strings.Compare(a.Path, b.Path)
	})

	return reports, nil
}
