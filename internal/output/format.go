package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
)

type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
	FormatCSV   Format = "csv"
)

// Write serializes raw JSON to w using the requested format.
// table and csv require raw to be an array of objects; returns an error otherwise.
func Write(w io.Writer, raw json.RawMessage, f Format, compact bool) error {
	switch f {
	case FormatTable:
		return writeTable(w, raw)
	case FormatCSV:
		return writeCSV(w, raw)
	default:
		return writeJSON(w, raw, compact)
	}
}

func writeJSON(w io.Writer, raw json.RawMessage, compact bool) error {
	if compact {
		var buf bytes.Buffer
		if err := json.Compact(&buf, raw); err != nil {
			return err
		}
		buf.WriteByte('\n')
		_, err := buf.WriteTo(w)
		return err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return err
	}
	buf.WriteByte('\n')
	_, err := buf.WriteTo(w)
	return err
}

func toObjectSlice(raw json.RawMessage) ([]map[string]any, error) {
	var arr []any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("table/csv requires an array of objects; use --jq to extract an array first")
	}
	if len(arr) == 0 {
		return nil, nil
	}
	rows := make([]map[string]any, 0, len(arr))
	for i, el := range arr {
		m, ok := el.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("table/csv requires all array elements to be objects; element %d is not an object", i)
		}
		rows = append(rows, m)
	}
	return rows, nil
}

func collectKeys(rows []map[string]any) []string {
	seen := make(map[string]struct{})
	for _, row := range rows {
		for k := range row {
			seen[k] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func cellValue(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%g", t)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func writeTable(w io.Writer, raw json.RawMessage) error {
	rows, err := toObjectSlice(raw)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	keys := collectKeys(rows)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(keys, "\t"))
	for _, row := range rows {
		vals := make([]string, len(keys))
		for i, k := range keys {
			vals[i] = sanitizeTerminal(cellValue(row[k]))
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	return tw.Flush()
}

func writeCSV(w io.Writer, raw json.RawMessage) error {
	rows, err := toObjectSlice(raw)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	keys := collectKeys(rows)
	cw := csv.NewWriter(w)
	if err := cw.Write(keys); err != nil {
		return err
	}
	for _, row := range rows {
		vals := make([]string, len(keys))
		for i, k := range keys {
			vals[i] = neutralizeCSVFormula(cellValue(row[k]))
		}
		if err := cw.Write(vals); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func sanitizeTerminal(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return ' '
		}
		return r
	}, s)
}

func neutralizeCSVFormula(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	default:
		return s
	}
}
