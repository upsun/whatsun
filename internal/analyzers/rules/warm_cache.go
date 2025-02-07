package rules

import (
	"os"

	"what"
	"what/internal/eval"
)

// WarmCache can be used externally to generate a file containing cached expressions.
func WarmCache(filename string) error {
	cache, err := eval.NewFileCache(filename)
	if err != nil {
		return err
	}

	root := "."
	celOptions := defaultEnvOptions(os.DirFS("."), &root)
	ev, err := eval.NewEvaluator(&eval.Config{Cache: cache, EnvOptions: celOptions})
	if err != nil {
		return err
	}
	for _, rs := range what.Config {
		for _, r := range rs.Rules {
			if r.When != "" {
				if _, err := ev.CompileAndCache(r.When); err != nil {
					return err
				}
			}
		}
	}

	return cache.Save()
}
