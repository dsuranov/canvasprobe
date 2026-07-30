package jsonpipe

import (
	"encoding/json"
)

// ApplyFields keeps selected top-level keys in each object of an array.
// It deliberately rejects nested envelopes; callers should use jq for trees.
func ApplyFields(raw json.RawMessage, fields []string) (json.RawMessage, error) {
	if len(fields) == 0 {
		return raw, nil
	}
	allow := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		allow[f] = struct{}{}
	}

	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	filtered := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out := make(map[string]any, len(allow))
		for k, val := range row {
			if _, ok := allow[k]; ok {
				out[k] = val
			}
		}
		filtered = append(filtered, out)
	}
	return json.Marshal(filtered)
}
