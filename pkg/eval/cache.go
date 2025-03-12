package eval

import (
	"sync"

	"github.com/google/cel-go/cel"
)

type Cache interface {
	Get(expr string) (ast *cel.Ast, ok bool)
	Set(expr string, ast *cel.Ast) error
}

type memoryCache struct {
	cache sync.Map
}

func (c *memoryCache) Get(expr string) (*cel.Ast, bool) {
	if a, ok := c.cache.Load(expr); ok {
		return a.(*cel.Ast), true //nolint:errcheck // the cached type is known
	}
	return nil, false
}

func (c *memoryCache) Set(expr string, ast *cel.Ast) error {
	c.cache.Store(expr, ast)
	return nil
}
