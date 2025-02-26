// Package eval provides an Evaluator to run CEL expressions.
package eval

import (
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
)

type Config struct {
	EnvOptions []cel.EnvOption
	Cache      Cache
}

// Evaluator supports running and optionally caching CEL expressions.
type Evaluator struct {
	celEnv       *cel.Env
	cache        Cache
	programCache sync.Map
}

func NewEvaluator(cnf *Config) (*Evaluator, error) {
	celEnv, err := cel.NewEnv(cnf.EnvOptions...)
	if err != nil {
		return nil, err
	}
	cache := cnf.Cache
	if cache == nil {
		cache = &memoryCache{}
	}

	return &Evaluator{celEnv: celEnv, cache: cache}, nil
}

func (e *Evaluator) Eval(expr string, input any) (ref.Val, error) {
	var ast *cel.Ast
	if e.cache != nil {
		if cached, ok := e.cache.Get(expr); ok {
			ast = cached
		}
	}
	if ast == nil {
		a, err := e.CompileAndCache(expr)
		if err != nil {
			return nil, err
		}
		ast = a
	}

	return e.eval(ast, input)
}

func (e *Evaluator) CompileAndCache(expr string) (*cel.Ast, error) {
	a, iss := e.celEnv.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	if e.cache != nil {
		if err := e.cache.Set(expr, a); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (e *Evaluator) eval(ast *cel.Ast, input any) (ref.Val, error) {
	var prg cel.Program
	if v, ok := e.programCache.Load(ast); ok {
		prg, _ = v.(cel.Program)
	} else {
		var err error
		prg, err = e.celEnv.Program(ast)
		if err != nil {
			return nil, err
		}
		e.programCache.Store(ast, prg)
	}

	out, _, err := prg.Eval(input)
	return out, err
}
