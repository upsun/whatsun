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

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/google/cel-go/common/types"
	"golang.org/x/sync/errgroup"

	"what/internal/eval"
	"what/internal/eval/celfuncs"
	"what/internal/fsgitignore"
)

type Analyzer struct {
	config    map[string]Ruleset
	evaluator *eval.Evaluator
	ignore    []string
}

func NewAnalyzer(ignore []string) (*Analyzer, error) {
	cache, err := eval.NewFileCacheWithContent(exprCache, "")
	if err != nil {
		return nil, err
	}
	ev, err := eval.NewEvaluator(&eval.Config{Cache: cache, EnvOptions: DefaultEnvOptions()})
	if err != nil {
		return nil, err
	}
	return &Analyzer{evaluator: ev, config: Config, ignore: ignore}, nil
}

func (a *Analyzer) Analyze(ctx context.Context, fsys fs.FS, root string) (Results, error) {
	dirs, err := a.collectDirectories(ctx, fsys, root)
	if err != nil {
		return nil, err
	}

	var results = make(Results, len(a.config))
	for name, rs := range a.config {
		res, err := a.applyRuleset(&rs, fsys, dirs)
		if err != nil {
			return nil, err
		}
		results[name] = Result{Paths: res}
	}

	return results, nil
}

func (a *Analyzer) collectDirectories(_ context.Context, fsys fs.FS, root string) ([]string, error) {
	var ignorePatterns = defaultIgnorePatterns
	if len(a.ignore) > 0 {
		ignorePatterns = append(ignorePatterns, fsgitignore.ParsePatterns(a.ignore, fsgitignore.Split(root))...)
	}
	var gitIgnorePatterns []gitignore.Pattern
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
		if gitignore.NewMatcher(append(gitIgnorePatterns, ignorePatterns...)).Match(fsgitignore.Split(path), true) {
			return fs.SkipDir
		}
		patterns, err := fsgitignore.ParseIgnoreFiles(fsys, path)
		if err != nil {
			return err
		}
		gitIgnorePatterns = append(gitIgnorePatterns, patterns...)
		directoryPaths = append(directoryPaths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return directoryPaths, nil
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

// TODO: only use defaults if no gitignore files are in the parent tree
var defaultIgnorePatterns = fsgitignore.ParsePatterns([]string{
	// IDE directories
	".idea/",
	".vscode/",
	".vs/",

	// Local development tool directories
	"/.ddev",

	// Build tool directories
	".build/",
	"bower_components",
	"elm-stuff/",
	".workspace/",
	"node_modules/",
	".next",
	".nuxt",

	// Tests and fixtures
	"tests/",
	"testdata/",
	"fixtures/",
	"Fixtures/",
	"__fixtures__/",

	// Python
	"__pycache__/",
	"venv/",
	"virtualenv/",
	".virtualenv/",

	// CI config
	".github/",
	".gitlab/",

	// Version control (".git" is already excluded)
	".hg/",
	".svn/",
	".bzr/",

	// Misc.
	".cache/",
	"_asm/",

	// TODO remove this when it can be parsed from e.g. composer.json
	"vendor/",
}, nil)

func (a *Analyzer) applyRuleset(rs *Ruleset, fsys fs.FS, directoryPaths []string) (map[string][]Report, error) {
	matcher := &Matcher{rs.Rules}

	var dirReports []dirReport

	eg := errgroup.Group{}
	eg.SetLimit(runtime.GOMAXPROCS(0))
	for _, d := range directoryPaths {
		eg.Go(func() error {
			input := celfuncs.FilesystemInput(fsys, d)
			evalWithInput := a.evalWithInput(input)
			dirSplit := fsgitignore.Split(d)
			matches, err := matcher.Match(func(rule *Rule) (bool, error) {
				// TODO tidy this up
				if len(rule.Ignore) > 0 {
					if gitignore.NewMatcher(fsgitignore.ParsePatterns(rule.Ignore, []string{})).Match(dirSplit, true) {
						return false, nil
					}
				}

				return evalWithInput(rule.When)
			})
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

	return reports, nil
}
