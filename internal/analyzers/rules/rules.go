package rules

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"what"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"

	"what/internal/eval"
	"what/internal/eval/celfuncs"
	"what/internal/match"
)

//go:embed expr.cache
var exprCache []byte

type Analyzer struct {
	config map[string]what.Ruleset

	AllowNested bool
	MaxDepth    int
}

func NewAnalyzer() (*Analyzer, error) {
	return &Analyzer{config: what.Config, AllowNested: false, MaxDepth: 3}, nil
}

func (*Analyzer) String() string {
	return "rules"
}

func defaultEnvOptions(fsys fs.FS, root *string) []cel.EnvOption {
	var celOptions []cel.EnvOption
	celOptions = append(celOptions, celfuncs.AllFileFunctions(&fsys, root)...)
	celOptions = append(celOptions, celfuncs.AllPackageManagerFunctions(&fsys, root)...)

	return append(
		celOptions,
		celfuncs.JSONQueryStringCELFunction(),
		celfuncs.VersionParse(),
	)
}

func (a *Analyzer) Analyze(_ context.Context, fsys fs.FS) (what.Result, error) {
	cache, err := eval.NewFileCacheWithContent(exprCache, "")
	if err != nil {
		return nil, err
	}
	dot := "."
	evRoot := &dot
	celOptions := defaultEnvOptions(fsys, evRoot)

	ev, err := eval.NewEvaluator(&eval.Config{Cache: cache, EnvOptions: celOptions})
	if err != nil {
		return nil, err
	}

	var results = make(Results, len(a.config))
	for name, rs := range a.config {
		res, err := a.applyRuleset(&rs, fsys, ev, evRoot)
		if err != nil {
			return nil, err
		}
		results[name] = res
	}

	return results, err
}

type Results map[string]*Result

func (r Results) String() string {
	if r == nil {
		return "[no results]"
	}

	names := make([]string, 0, len(r))
	for name := range r {
		names = append(names, name)
	}
	sort.Strings(names)

	s := ""
	for _, name := range names {
		s += fmt.Sprintf("\nRuleset: %s", name)
		res := r[name]
		if len(res.Directories) == 0 {
			s += "\n[No results]\n"
			continue
		}
		s += "\nPath\tMatches\n"
		lines := make([]string, 0, len(res.Directories))
		for dir, matches := range res.Directories {
			lines = append(lines, fmt.Sprintf("%s\t%+v", dir, matches))
		}
		sort.Strings(lines)
		s += strings.Join(lines, "\n")
		s += "\n"
	}

	return strings.TrimRight(s, "\n")
}

type Result struct {
	Directories map[string][]match.Match
}

func (a *Analyzer) applyRuleset(rs *what.Ruleset, fsys fs.FS, ev *eval.Evaluator, evRoot *string) (*Result, error) {
	var (
		result  = &Result{Directories: make(map[string][]match.Match)}
		evFunc  = evalFunc(ev)
		matcher = &match.Matcher{Rules: rs.Rules, Report: reportFunc(ev)}
	)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
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
			*evRoot = path
			m, err := matcher.Match(evFunc)
			if err != nil {
				return fmt.Errorf("in directory %s: %w", path, err)
			}
			if len(m) > 0 {
				result.Directories[path] = append(result.Directories[path], m...)
			}
			if !a.AllowNested && depth > 0 {
				return filepath.SkipDir
			}
		}

		if depth > a.MaxDepth {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

type Report struct {
	When string
	With map[string]string
}

func reportFunc(ev *eval.Evaluator) func(rules []*what.Rule) any {
	return func(rules []*what.Rule) any {
		var reports []Report
		for _, rule := range rules {
			rep := Report{When: rule.When}
			if len(rule.With) == 0 {
				reports = append(reports, rep)
				continue
			}
			rep.With = make(map[string]string)
			for name, expr := range rule.With {
				val, err := ev.Eval(expr)
				if err != nil {
					rep.With[name] = fmt.Sprint("[ERROR] ", err.Error())
					continue
				}
				rep.With[name] = fmt.Sprint(val)
			}
			reports = append(reports, rep)
		}
		slices.SortFunc(reports, func(a, b Report) int {
			return strings.Compare(a.When, b.When)
		})
		return reports
	}
}

func evalFunc(ev *eval.Evaluator) func(string) (bool, error) {
	return func(condition string) (bool, error) {
		val, err := ev.Eval(condition)
		if err != nil {
			return false, err
		}
		switch val.(type) {
		case types.Bool:
			return bool(val.(types.Bool)), nil
		case types.String:
			return string(val.(types.String)) != "", nil
		case *types.Optional:
			return val.(*types.Optional).HasValue(), nil
		case types.Null:
			return false, nil
		}
		return false, errors.New("condition returns unexpected type")
	}
}
