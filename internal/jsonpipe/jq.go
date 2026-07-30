package jsonpipe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// ApplyJQ runs expr against raw using gojq. The operation is bounded by ctx
// (caller should set a 5-second deadline to guard against infinite expressions).
// The result is marshaled back to JSON; multiple output values are wrapped in an array.
func ApplyJQ(ctx context.Context, raw json.RawMessage, expr string) (json.RawMessage, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("jq parse: %w", err)
	}

	var input any
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, err
	}

	iter := query.RunWithContext(ctx, input)

	var results []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("jq timeout: %w", ctx.Err())
			}
			return nil, fmt.Errorf("jq: %w", err)
		}
		results = append(results, v)
	}

	if len(results) == 1 {
		return json.Marshal(results[0])
	}
	return json.Marshal(results)
}
