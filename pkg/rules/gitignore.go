package rules

import (
	"sync"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"

	"github.com/upsun/whatsun/internal/fsgitignore"
)

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

type Ignorer interface {
	GetIgnores() []string
}

var ignoreMatcherCache sync.Map

func getIgnoreMatcher(i Ignorer) gitignore.Matcher {
	ignores := i.GetIgnores()
	if len(ignores) == 0 {
		return nil
	}
	if m, ok := ignoreMatcherCache.Load(i); ok {
		return m.(gitignore.Matcher)
	}
	m := gitignore.NewMatcher(fsgitignore.ParsePatterns(ignores, []string{}))
	ignoreMatcherCache.Store(i, m)
	return m
}
