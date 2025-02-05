package rules

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

//go:embed rule_sets.yml
var configData []byte

type Analyzer struct {
	config match.Config

	AllowNested bool
	MaxDepth    int
}

func NewAnalyzer() (*Analyzer, error) {
	cnf, err := match.ParseConfig(bytes.NewReader(configData))
	if err != nil {
		return nil, err
	}

	return &Analyzer{config: cnf, AllowNested: false, MaxDepth: 3}, nil
}

func (*Analyzer) String() string {
	return "rules"
}

func (a *Analyzer) Analyze(_ context.Context, fsys fs.FS) (what.Result, error) {
	cache, err := eval.NewFileCacheWithContent(exprCache, "")
	if err != nil {
		return nil, err
	}
	dot := "."
	evRoot := &dot
	var celOptions []cel.EnvOption
	celOptions = append(celOptions, celfuncs.AllFileFunctions(&fsys, evRoot)...)
	celOptions = append(celOptions, celfuncs.AllComposerFunctions(&fsys, evRoot)...)
	celOptions = append(
		celOptions,
		celfuncs.JSONQueryStringCELFunction(),
		celfuncs.VersionParse(),
	)

	ev, err := eval.NewEvaluator(&eval.Config{
		Cache:      cache,
		EnvOptions: append(celOptions, celfuncs.AllComposerFunctions(&fsys, evRoot)...),
	})
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
	s := ""
	for name, rs := range r {
		s += fmt.Sprintf("\nRuleset: %s", name)
		if len(rs.Directories) == 0 {
			s += name + ": [no results]\n"
			continue
		}
		s += "\n\nPath\tMatches\n"
		for dir, matches := range rs.Directories {
			s += fmt.Sprintf("%s\t%s\n", dir, matches)
		}
		s += "\n"
	}
	return strings.TrimRight(s, "\n")
}

type Result struct {
	Directories map[string][]match.Match
}

func (a *Analyzer) applyRuleset(rs *match.RuleSet, fsys fs.FS, ev *eval.Evaluator, evRoot *string) (*Result, error) {
	var (
		result  = &Result{Directories: make(map[string][]match.Match)}
		evFunc  = evalFunc(ev)
		matcher = &match.Matcher{Rules: rs.Rules}
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
				return err
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
