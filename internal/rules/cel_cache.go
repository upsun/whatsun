package rules

import (
	_ "embed"
	"os"

	"what/internal/eval"
)

//go:embed expr.cache
var exprCache []byte

// WarmCache can be used externally to generate a file containing cached expressions.
func WarmCache(filename string) error {
	cache, err := eval.NewFileCacheWithContent(nil, filename)
	if err != nil {
		return err
	}

	root := "."
	celOptions := defaultEnvOptions(os.DirFS("."), &root)
	ev, err := eval.NewEvaluator(&eval.Config{Cache: cache, EnvOptions: celOptions})
	if err != nil {
		return err
	}
	for _, rs := range Config {
		for _, r := range rs.Rules {
			if r.When != "" {
				if _, err := ev.CompileAndCache(r.When); err != nil {
					return err
				}
			}
			for _, expr := range r.With {
				if _, err := ev.CompileAndCache(expr); err != nil {
					return err
				}
			}
		}
	}

	return cache.Save()
}
