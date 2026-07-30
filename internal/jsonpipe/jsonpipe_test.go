package jsonpipe

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestApplyFields(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		fields []string
		want   string
	}{
		{
			name:   "array of objects",
			input:  `[{"id":"1","name":"A","nested":{"keep":true}},{"id":"2","name":"B"}]`,
			fields: []string{"id", "nested"},
			want:   `[{"id":"1","nested":{"keep":true}},{"id":"2"}]`,
		},
		{
			name:   "empty fields returns original",
			input:  `[{"id":"1","name":"A"}]`,
			fields: nil,
			want:   `[{"id":"1","name":"A"}]`,
		},
		{
			name:   "empty array",
			input:  `[]`,
			fields: []string{"id"},
			want:   `[]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ApplyFields(json.RawMessage(tc.input), tc.fields)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tc.want, string(got))
		})
	}
}

func TestApplyFieldsRejectsEnvelope(t *testing.T) {
	if _, err := ApplyFields(json.RawMessage(`{"document":{"id":"0:0"}}`), []string{"id"}); err == nil {
		t.Fatal("expected envelope to be rejected")
	}
}

func TestApplyJQ(t *testing.T) {
	cases := []struct {
		name  string
		input string
		expr  string
		want  string
	}{
		{"identity", `{"id":"1"}`, ".", `{"id":"1"}`},
		{"field access", `{"id":"1","name":"a"}`, ".name", `"a"`},
		{"array iteration", `[{"id":"1"},{"id":"2"}]`, ".[].id", `["1","2"]`},
		{"filter", `[{"id":"1","v":10},{"id":"2","v":5}]`, "[.[] | select(.v > 7)]", `[{"id":"1","v":10}]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ApplyJQ(context.Background(), json.RawMessage(tc.input), tc.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tc.want, string(got))
		})
	}
}

func TestApplyJQTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := ApplyJQ(ctx, json.RawMessage(`null`), "range(0;1e9)")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}

func TestApplyJQParseError(t *testing.T) {
	if _, err := ApplyJQ(context.Background(), json.RawMessage(`{}`), "!!!invalid"); err == nil {
		t.Fatal("expected parse error")
	}
}

func assertJSONEqual(t *testing.T, want, got string) {
	t.Helper()
	var wv, gv any
	if err := json.Unmarshal([]byte(want), &wv); err != nil {
		t.Fatalf("invalid want JSON %q: %v", want, err)
	}
	if err := json.Unmarshal([]byte(got), &gv); err != nil {
		t.Fatalf("invalid got JSON %q: %v", got, err)
	}
	wBytes, _ := json.Marshal(wv)
	gBytes, _ := json.Marshal(gv)
	if string(wBytes) != string(gBytes) {
		t.Errorf("JSON mismatch:\n  want: %s\n   got: %s", wBytes, gBytes)
	}
}
