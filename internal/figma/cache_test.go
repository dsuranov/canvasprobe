package figma

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const testOrigin = "https://api.figma.com"

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	c, err := NewCache(t.TempDir(), 5*time.Minute)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	return c
}

func TestCacheRoundTrip(t *testing.T) {
	c := newTestCache(t)
	want := `{"id":"1"}`
	c.Set("GET", testOrigin, "/v1/me", "", "tok123", []byte(want))
	got, ok := c.Get("GET", testOrigin, "/v1/me", "", "tok123")
	if !ok || string(got) != want {
		t.Fatalf("got %q hit=%v", got, ok)
	}
}

func TestCacheTTLExpire(t *testing.T) {
	c := newTestCache(t)
	now := time.Now()
	c.nowFn = func() time.Time { return now }
	c.Set("GET", testOrigin, "/v1/me", "", "tok123", []byte(`{"id":"1"}`))
	c.nowFn = func() time.Time { return now.Add(c.ttl + time.Second) }
	if _, ok := c.Get("GET", testOrigin, "/v1/me", "", "tok123"); ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestCacheCorruptedFile(t *testing.T) {
	c := newTestCache(t)
	c.Set("GET", testOrigin, "/v1/me", "", "tok123", []byte(`{"id":"1"}`))
	h := c.entryKey("GET", testOrigin, "/v1/me", "", "tok123")
	fpath := filepath.Join(c.dir, fmt.Sprintf("%x.json", h))
	if err := os.WriteFile(fpath, []byte("not json{{{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get("GET", testOrigin, "/v1/me", "", "tok123"); ok {
		t.Error("expected cache miss")
	}
	if _, err := os.Stat(fpath); !os.IsNotExist(err) {
		t.Error("expected corrupted file to be deleted")
	}
}

func TestCacheAtomicWriteRace(t *testing.T) {
	c := newTestCache(t)
	data := []byte(`{"id":"1"}`)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Set("GET", testOrigin, "/v1/me", "", "tok123", data)
		}()
	}
	wg.Wait()
	got, ok := c.Get("GET", testOrigin, "/v1/me", "", "tok123")
	if !ok || !json.Valid(got) {
		t.Fatal("cache entry missing or corrupted")
	}
}

func TestCacheFullTokenAndOriginIsolation(t *testing.T) {
	c := newTestCache(t)
	tokenA := "samepref-A"
	tokenB := "samepref-B"
	c.Set("GET", testOrigin, "/v1/me", "", tokenA, []byte(`{"id":"A"}`))
	c.Set("GET", testOrigin, "/v1/me", "", tokenB, []byte(`{"id":"B"}`))
	c.Set("GET", "https://api.example.test", "/v1/me", "", tokenA, []byte(`{"id":"C"}`))

	gotA, okA := c.Get("GET", testOrigin, "/v1/me", "", tokenA)
	gotB, okB := c.Get("GET", testOrigin, "/v1/me", "", tokenB)
	gotC, okC := c.Get("GET", "https://api.example.test", "/v1/me", "", tokenA)
	if !okA || !okB || !okC {
		t.Fatal("expected cache hits")
	}
	if string(gotA) == string(gotB) || string(gotA) == string(gotC) {
		t.Fatal("token or origin collision")
	}
}

func TestCachePermissionsStatusAndPurge(t *testing.T) {
	c := newTestCache(t)
	c.Set("GET", testOrigin, "/v1/me", "", "token", []byte(`{"id":"1"}`))
	info, err := os.Stat(c.dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0700 {
		t.Fatalf("cache dir mode=%o", info.Mode().Perm())
	}
	stats, err := c.Status()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Entries != 1 || stats.Bytes == 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	removed, err := c.Purge()
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed=%d", removed)
	}
}

func TestCacheGC(t *testing.T) {
	c := newTestCache(t)
	c.ttl = 100 * time.Millisecond
	now := time.Now()
	c.nowFn = func() time.Time { return now }
	c.Set("GET", testOrigin, "/v1/old", "", "tok", []byte(`{"id":"1"}`))
	c.nowFn = func() time.Time { return now.Add(11 * c.ttl) }
	c.gc()
	h := c.entryKey("GET", testOrigin, "/v1/old", "", "tok")
	if _, err := os.Stat(filepath.Join(c.dir, fmt.Sprintf("%x.json", h))); !os.IsNotExist(err) {
		t.Error("expected GC to delete expired file")
	}
}
