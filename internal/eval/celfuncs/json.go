package celfuncs

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
)

func JQ() cel.EnvOption {
	FuncDocs["jq"] = FuncDoc{
		Comment: "Query JSON bytes (e.g. file contents) using JQ",
		Args: []ArgDoc{
			{"contents", ""},
			{"query", ""},
		},
	}

	return bytesStringReturnsStringErr("jq", func(b []byte, expr string) (string, error) {
		m := map[string]any{}
		if err := json.Unmarshal(b, &m); err != nil {
			return "", err
		}
		return jq(m, expr)
	})
}

func YQ() cel.EnvOption {
	FuncDocs["yq"] = FuncDoc{
		Comment: "Query YAML bytes (e.g. file contents) using YQ (same syntax as JQ)",
		Args: []ArgDoc{
			{"contents", ""},
			{"query", ""},
		},
	}

	return bytesStringReturnsStringErr("yq", func(b []byte, expr string) (string, error) {
		m := map[string]any{}
		if err := yaml.Unmarshal(b, &m); err != nil {
			return "", err
		}
		return jq(m, expr)
	})
}

func jq(m map[string]any, expr string) (string, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return "", err
	}
	// TODO use context?
	iter := query.Run(m)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			var he *gojq.HaltError
			if errors.As(err, &he) && he.Value() == nil {
				break
			}
			return "", err
		}
		if v == nil {
			return "", nil
		}
		return fmt.Sprint(v), nil //lint:ignore SA4004 false positive
	}

	return "", errors.New("failed to evaluate JSON query")
}
