package main

import (
	"fmt"
	"os"

	"what/internal/config"
	"what/internal/eval"
	"what/internal/eval/celfuncs"
	"what/internal/rules"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: warm_cache <filename>")
		os.Exit(1)
	}
	if err := warmCache(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "Cache saved to:", os.Args[1])
}

func warmCache(filename string) error {
	rulesets, err := config.LoadEmbeddedRulesets()
	if err != nil {
		return err
	}

	cache, err := eval.NewFileCacheWithContent(nil, filename)
	if err != nil {
		return err
	}

	ev, err := eval.NewEvaluator(&eval.Config{Cache: cache, EnvOptions: celfuncs.DefaultEnvOptions()})
	if err != nil {
		return err
	}
	for _, rs := range rulesets {
		for _, r := range rs.GetRules() {
			if condition := r.GetCondition(); condition != "" {
				if _, err := ev.CompileAndCache(condition); err != nil {
					return err
				}
			}
			if wm, ok := r.(rules.WithMetadata); ok {
				for _, expr := range wm.GetMetadata() {
					if _, err := ev.CompileAndCache(expr); err != nil {
						return err
					}
				}
			}
		}
	}

	return cache.Save()
}
