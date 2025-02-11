// Package rules applies rules and combines the results.
package rules

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"what/internal/eval"
)

type Analyzer struct {
	config map[string]Ruleset
	cache  eval.Cache
}

func NewAnalyzer() (*Analyzer, error) {
	cache, err := eval.NewFileCacheWithContent(exprCache, "")
	if err != nil {
		return nil, err
	}
	return &Analyzer{cache: cache, config: Config}, nil
}

func (a *Analyzer) Analyze(_ context.Context, fsys fs.FS) (Results, error) {
	dot := "."
	evRoot := &dot
	celOptions := DefaultEnvOptions(fsys, evRoot)

	ev, err := eval.NewEvaluator(&eval.Config{Cache: a.cache, EnvOptions: celOptions})
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

type Results map[string]Result

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
	Directories map[string][]Match
}

func (a *Analyzer) applyRuleset(rs *Ruleset, fsys fs.FS, ev *eval.Evaluator, evRoot *string) (Result, error) {
	var (
		result  = Result{Directories: make(map[string][]Match)}
		evFunc  = evalFunc(ev)
		matcher = &Matcher{Rules: rs.Rules, Report: reportFunc(ev)}
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
			if rs.MaxNestedDepth != 0 && depth >= rs.MaxNestedDepth {
				return filepath.SkipDir
			}
		}

		if depth >= rs.MaxDepth {
			return filepath.SkipDir
		}
		return nil
	})

	return result, err
}

type Report struct {
	Rule string
	With map[string]string
}
