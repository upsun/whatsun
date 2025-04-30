package rules

import (
	_ "embed"
	"sync"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"

	"github.com/upsun/whatsun/internal/fsgitignore"
)

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
		return m.(gitignore.Matcher) //nolint:errcheck // the cached type is known
	}
	m := gitignore.NewMatcher(fsgitignore.ParsePatterns(ignores, []string{}))
	ignoreMatcherCache.Store(i, m)
	return m
}
