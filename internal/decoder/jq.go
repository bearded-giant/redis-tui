package decoder

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
)

// ApplyJqPath runs a jq expression over the JSON-rendered form of decoded
// and returns the result as a pretty-printed JSON string. Multi-result
// expressions (e.g. `.[] | .id`) are returned as a JSON array. An error in
// the expression itself is wrapped, an empty result set returns "null".
func ApplyJqPath(decoded Decoded, expr string) (string, error) {
	if expr == "" {
		return decoded.Pretty, nil
	}
	if decoded.Pretty == "" {
		return "", errors.New("no decoded JSON to filter")
	}

	var input any
	if err := json.Unmarshal([]byte(decoded.Pretty), &input); err != nil {
		return "", fmt.Errorf("decoded value is not JSON: %w", err)
	}

	query, err := gojq.Parse(expr)
	if err != nil {
		return "", fmt.Errorf("jq parse: %w", err)
	}

	iter := query.Run(input)
	var results []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if jqErr, isErr := v.(error); isErr {
			return "", fmt.Errorf("jq eval: %w", jqErr)
		}
		results = append(results, v)
	}

	var out any
	switch len(results) {
	case 0:
		out = nil
	case 1:
		out = results[0]
	default:
		out = results
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(b), nil
}
