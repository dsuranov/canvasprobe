package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompactJSONActuallyCompacts(t *testing.T) {
	var out bytes.Buffer
	raw := []byte("{\n  \"a\": 1,\n  \"b\": [2, 3]\n}\n")
	if err := Write(&out, raw, FormatJSON, true); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "{\"a\":1,\"b\":[2,3]}\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTableRemovesTerminalControls(t *testing.T) {
	var out bytes.Buffer
	raw := []byte(`[{"name":"safe\u001b]0;owned\u0007text"}]`)
	if err := Write(&out, raw, FormatTable, false); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(out.String(), "\x1b\x07") {
		t.Fatalf("terminal controls remained: %q", out.String())
	}
}

func TestCSVNeutralizesSpreadsheetFormulas(t *testing.T) {
	var out bytes.Buffer
	raw := []byte(`[{"name":"=HYPERLINK(\"https://example.test\")"},{"name":"safe"}]`)
	if err := Write(&out, raw, FormatCSV, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `'=HYPERLINK`) {
		t.Fatalf("formula was not neutralized: %q", out.String())
	}
}
