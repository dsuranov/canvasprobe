package figma

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := New(Options{
		Token:      "test-token",
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.jitterFn = func() time.Duration { return 0 }
	c.sleepFn = func(time.Duration) {}
	return c
}

func TestGetMeHeadersAndBody(t *testing.T) {
	want := `{"id":"1","email":"test@example.com"}`
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Figma-Token") != "test-token" {
			t.Error("missing token header")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("missing Accept header")
		}
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "fgm-c/") {
			t.Errorf("unexpected User-Agent: %q", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte(want))
	}))
	data, err := c.GetMe(context.Background())
	if err != nil || string(data) != want {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestHTTPStatusClassification(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusNotFound, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			_, err := c.GetMe(context.Background())
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestGetRetries429And5xxOnce(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					if status == http.StatusTooManyRequests {
						w.Header().Set("Retry-After", "0")
					}
					w.WriteHeader(status)
					return
				}
				_, _ = w.Write([]byte(`{"id":"1"}`))
			}))
			if _, err := c.GetMe(context.Background()); err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 2 {
				t.Fatalf("calls=%d", calls.Load())
			}
		})
	}
}

func TestRetryAfterTooLongDoesNotRetry(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	_, err := c.GetMe(context.Background())
	if !errors.Is(err, ErrRateLimited) || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestResponseTooLarge(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 101)))
	}))
	c.maxBody = 100
	_, err := c.GetMe(context.Background())
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("got %v", err)
	}
}

func TestRedirectDoesNotLeakTokenAcrossOrigins(t *testing.T) {
	var targetCalls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetCalls.Add(1)
		if r.Header.Get("X-Figma-Token") != "" {
			t.Error("token leaked to redirect target")
		}
	}))
	defer target.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/capture", http.StatusFound)
	}))
	defer source.Close()

	c, err := New(Options{Token: "secret", BaseURL: source.URL, HTTPClient: source.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetMe(context.Background()); err == nil || !strings.Contains(err.Error(), "redirect blocked") {
		t.Fatalf("expected redirect error, got %v", err)
	}
	if targetCalls.Load() != 0 {
		t.Fatalf("redirect target was called %d times", targetCalls.Load())
	}
}

func TestPostCommentPayloadAndNoRetry(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/v1/files/abc/comments" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["message"] != "hello" || body["comment_id"] != "42" {
			t.Fatalf("unexpected body: %#v", body)
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"err":"temporary"}`))
	}))
	_, err := c.PostComment(context.Background(), "abc", "hello", "42")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("write was retried: calls=%d", calls.Load())
	}
}

func TestDeleteComment(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/files/abc/comments/42:1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := c.DeleteComment(context.Background(), "abc", "42:1"); err != nil {
		t.Fatal(err)
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := New(Options{}); err == nil {
		t.Fatal("expected no-token error")
	}
	if _, err := New(Options{Token: "x", BaseURL: "http://example.com"}); err == nil {
		t.Fatal("expected custom-origin injection error")
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		header string
		want   time.Duration
		ok     bool
	}{
		{"", 0, false},
		{"abc", 0, false},
		{"0", 0, true},
		{"5", 5 * time.Second, true},
	}
	for _, tt := range tests {
		got, ok := parseRetryAfter(tt.header)
		if got != tt.want || ok != tt.ok {
			t.Errorf("%q: got (%v,%v)", tt.header, got, ok)
		}
	}
}
