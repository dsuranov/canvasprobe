package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dsuranov/fgm-c/internal/figma"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Figma-Token") == "" {
			http.Error(w, `{"err":"missing token"}`, http.StatusForbidden)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "u1", "handle": "testuser"})
	})
	mux.HandleFunc("/v1/files/abc123", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"document": map[string]any{
				"id":   "0:0",
				"name": "root",
				"type": "DOCUMENT",
				"children": []any{
					map[string]any{"id": "1:1", "name": "Page", "type": "CANVAS"},
				},
			},
			"name":    "Test File",
			"version": "1",
			"depth":   r.URL.Query().Get("depth"),
		})
	})
	mux.HandleFunc("/v1/files/abc123/nodes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"nodes": map[string]any{"1:2": map[string]any{
				"document": map[string]any{"id": "1:2", "name": "Frame1", "type": "FRAME"},
			}},
		})
	})
	mux.HandleFunc("/v1/files/abc123/components", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{"components": []any{
				map[string]any{"key": "cmpA", "name": "Button", "node_id": "1:2"},
			}},
		})
	})
	mux.HandleFunc("/v1/files/abc123/styles", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{"styles": []any{
				map[string]any{"key": "styA", "name": "Primary", "style_type": "FILL"},
			}},
		})
	})
	mux.HandleFunc("/v1/files/abc123/comments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"comments":[{"id":"c1","message":"review"}]}`))
		case http.MethodPost:
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "c2", "message": body["message"]})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/v1/files/abc123/comments/c2", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func runCmd(t *testing.T, srv *httptest.Server, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	t.Setenv("FGM_C_TOKEN", "testtoken")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	oldOverride := clientOptionsOverride
	clientOptionsOverride = func(opts *figma.Options) {
		opts.BaseURL = srv.URL
		opts.HTTPClient = srv.Client()
	}
	t.Cleanup(func() { clientOptionsOverride = oldOverride })

	root := newRootCmd()
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs(append([]string{"--no-cache"}, args...))
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestE2EMe(t *testing.T) {
	stdout, _, err := runCmd(t, testServer(t), "me")
	if err != nil || !strings.Contains(stdout, "testuser") {
		t.Fatalf("stdout=%q err=%v", stdout, err)
	}
}

func TestE2EFileDefaultDepthPreservesEnvelope(t *testing.T) {
	stdout, stderr, err := runCmd(t, testServer(t), "file", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Test File", `"document"`, `"children"`, `"depth": "2"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing %q in %s", want, stdout)
		}
	}
	if !strings.Contains(stderr, "applied default --depth 2") {
		t.Fatalf("missing warning: %q", stderr)
	}
}

func TestE2EFileFieldsRejected(t *testing.T) {
	_, _, err := runCmd(t, testServer(t), "file", "abc123", "--fields", "name")
	if err == nil || !strings.Contains(err.Error(), "supported only") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestE2EFileNodes(t *testing.T) {
	stdout, _, err := runCmd(t, testServer(t), "file", "abc123", "nodes", "1:2", "--depth", "2")
	if err != nil || !strings.Contains(stdout, "Frame1") {
		t.Fatalf("stdout=%q err=%v", stdout, err)
	}
}

func TestE2EComponentsFieldsAndCSV(t *testing.T) {
	stdout, _, err := runCmd(t, testServer(t), "--format", "csv", "file", "abc123", "components", "--fields", "name,key")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "Button") || strings.Contains(stdout, "node_id") {
		t.Fatalf("unexpected output: %s", stdout)
	}
}

func TestIncompatibleFlagsFail(t *testing.T) {
	srv := testServer(t)
	for _, args := range [][]string{
		{"--format", "xml", "me"},
		{"file", "abc123", "components", "--depth", "2"},
		{"file", "abc123", "nodes", "1:2", "--version", "v1"},
	} {
		if _, _, err := runCmd(t, srv, args...); err == nil {
			t.Fatalf("expected failure for %v", args)
		}
	}
}

func TestE2ECompactJSON(t *testing.T) {
	stdout, _, err := runCmd(t, testServer(t), "--compact", "me")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout, "\n  ") || !strings.HasSuffix(stdout, "\n") {
		t.Fatalf("not compact: %q", stdout)
	}
}

func TestE2EComments(t *testing.T) {
	srv := testServer(t)
	stdout, _, err := runCmd(t, srv, "comments", "list", "abc123")
	if err != nil || !strings.Contains(stdout, "review") {
		t.Fatalf("list stdout=%q err=%v", stdout, err)
	}
	stdout, _, err = runCmd(t, srv, "comments", "create", "abc123", "--message", "ship it")
	if err != nil || !strings.Contains(stdout, "ship it") {
		t.Fatalf("create stdout=%q err=%v", stdout, err)
	}
	if _, _, err = runCmd(t, srv, "comments", "delete", "abc123", "c2"); err == nil {
		t.Fatal("delete without --yes should fail")
	}
	stdout, _, err = runCmd(t, srv, "comments", "delete", "abc123", "c2", "--yes")
	if err != nil || !strings.Contains(stdout, `"deleted":true`) {
		t.Fatalf("delete stdout=%q err=%v", stdout, err)
	}
}

func TestTokenStdinValidation(t *testing.T) {
	if got, err := readToken(strings.NewReader(" token-123\n")); err != nil || got != "token-123" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	if _, err := readToken(strings.NewReader("two tokens")); err == nil {
		t.Fatal("expected whitespace error")
	}
}

func TestNoLegacyTokenOrBaseURLFlags(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"--token", "secret", "me"})
	if err := root.Execute(); err == nil {
		t.Fatal("legacy --token unexpectedly accepted")
	}
	root = newRootCmd()
	root.SetArgs([]string{"--base-url", "https://example.test", "me"})
	if err := root.Execute(); err == nil {
		t.Fatal("--base-url unexpectedly accepted")
	}
}
