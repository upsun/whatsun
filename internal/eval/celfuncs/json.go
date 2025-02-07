package celfuncs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/itchyny/gojq"
)

// JSONQueryStringCELFunction defines a CEL function `json.queryString(b, expr) -> string`.
func JSONQueryStringCELFunction() cel.EnvOption {
	FuncComments["json.queryString"] = "Query a byte string (e.g. file contents) using JQ"

	return bytesStringReturnsStringErr("json.queryString", jq)
}

func jq(b []byte, expr string) (string, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return "", err
	}
	m := map[string]any{}
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&m); err != nil {
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
		return fmt.Sprint(v), nil
	}

	return "", errors.New("failed to evaluate JSON query")
}
