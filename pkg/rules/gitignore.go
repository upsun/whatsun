package rules

import (
	"bytes"
	_ "embed"
	"sync"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"

	"github.com/upsun/whatsun/internal/fsgitignore"
)

//go:embed gitignore-defaults
var defaultIgnoreFile []byte

var defaultIgnorePatterns = fsgitignore.ParseIgnoreFile(bytes.NewReader(defaultIgnoreFile), nil)

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
