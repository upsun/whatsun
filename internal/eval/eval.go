// Package eval provides an Evaluator to run CEL expressions.
package eval

import (
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
)

type Config struct {
	EnvOptions []cel.EnvOption
	Cache      Cache
}

// Evaluator supports running and optionally caching CEL expressions.
type Evaluator struct {
	celEnv       *cel.Env
	cache        Cache
	programCache *programCache
}

func NewEvaluator(cnf *Config) (*Evaluator, error) {
	celEnv, err := newCelEnv(cnf)
	if err != nil {
		return nil, err
	}
	cache := cnf.Cache
	if cache == nil {
		cache = &memoryCache{}
	}

	return &Evaluator{celEnv: celEnv, cache: cache, programCache: &programCache{}}, nil
}

func (e *Evaluator) Eval(expr string) (ref.Val, error) {
	var ast *cel.Ast
	if e.cache != nil {
		if cached, ok := e.cache.Get(expr); ok {
			ast = cached
		}
	}
	if ast == nil {
		a, iss := e.celEnv.Compile(expr)
		if iss.Err() != nil {
			return nil, iss.Err()
		}
		ast = a
		if e.cache != nil {
			e.cache.Set(expr, ast)
		}
	}

	return e.eval(ast)
}

func (e *Evaluator) eval(ast *cel.Ast) (ref.Val, error) {
	prg, ok := e.programCache.get(ast)
	if !ok {
		var err error
		prg, err = e.celEnv.Program(ast)
		if err != nil {
			return nil, err
		}
		e.programCache.set(ast, prg)
	}

	out, _, err := prg.Eval(map[string]any{})
	return out, err
}

func newCelEnv(cnf *Config) (*cel.Env, error) {
	options := append(cnf.EnvOptions,
		ext.Lists(),
		ext.Strings(),
		ext.NativeTypes(),
	)

	celEnv, err := cel.NewEnv(options...)
	if err != nil {
		return nil, err
	}

	return celEnv, nil
}

type programCache struct {
	programs map[*cel.Ast]cel.Program
	mux      sync.RWMutex
}

func (pc *programCache) get(ast *cel.Ast) (cel.Program, bool) {
	pc.mux.RLock()
	defer pc.mux.RUnlock()
	if pc.programs != nil {
		prg, ok := pc.programs[ast]
		return prg, ok
	}
	return nil, false
}

func (pc *programCache) set(ast *cel.Ast, program cel.Program) {
	pc.mux.Lock()
	defer pc.mux.Unlock()
	if pc.programs == nil {
		pc.programs = make(map[*cel.Ast]cel.Program)
	}
	pc.programs[ast] = program
}
