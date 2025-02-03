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
	cache map[string]*cel.Ast
	mutex sync.RWMutex
}

func (c *memoryCache) Get(expr string) (*cel.Ast, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.cache != nil {
		if ast, ok := c.cache[expr]; ok {
			return ast, true
		}
	}
	return nil, false
}

func (c *memoryCache) Set(expr string, ast *cel.Ast) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.cache == nil {
		c.cache = make(map[string]*cel.Ast)
	}
	c.cache[expr] = ast
	return nil
}
