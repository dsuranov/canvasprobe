package figma

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Cache struct {
	dir   string
	ttl   time.Duration
	nowFn func() time.Time
}

func DefaultCacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "fgm-c")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "fgm-c")
}

func NewCache(dir string, ttl time.Duration) (*Cache, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("cache: mkdir %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return nil, fmt.Errorf("cache: chmod %s: %w", dir, err)
	}
	return &Cache{dir: dir, ttl: ttl, nowFn: time.Now}, nil
}

func (c *Cache) entryKey(method, origin, path, query, tok string) [32]byte {
	tokenHash := sha256.Sum256([]byte(tok))
	material := method + "\x00" + origin + "\x00" + path + "\x00" + query + "\x00" + fmt.Sprintf("%x", tokenHash)
	return sha256.Sum256([]byte(material))
}

func (c *Cache) Get(method, origin, path, query, tok string) ([]byte, bool) {
	h := c.entryKey(method, origin, path, query, tok)
	fpath := filepath.Join(c.dir, fmt.Sprintf("%x.json", h))

	info, err := os.Stat(fpath)
	if err != nil {
		return nil, false
	}
	if c.nowFn().Sub(info.ModTime()) > c.ttl {
		return nil, false
	}

	data, err := os.ReadFile(fpath)
	if err != nil {
		return nil, false
	}
	if !json.Valid(data) {
		os.Remove(fpath)
		return nil, false
	}
	return data, true
}

func (c *Cache) Set(method, origin, path, query, tok string, data []byte) {
	h := c.entryKey(method, origin, path, query, tok)
	fpath := filepath.Join(c.dir, fmt.Sprintf("%x.json", h))

	tmp, err := os.CreateTemp(c.dir, "*.tmp")
	if err != nil {
		return
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return
	}
	if err := os.Rename(tmpPath, fpath); err != nil {
		os.Remove(tmpPath)
		return
	}

	// Lazy GC: ~1% of writes (3 out of 256 first-byte values divisible by 100)
	if h[0]%100 == 0 {
		c.gc()
	}
}

type CacheStats struct {
	Entries int
	Bytes   int64
}

func (c *Cache) Status() (CacheStats, error) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return CacheStats{}, fmt.Errorf("cache: read dir: %w", err)
	}
	var stats CacheStats
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return CacheStats{}, fmt.Errorf("cache: stat %s: %w", entry.Name(), err)
		}
		stats.Entries++
		stats.Bytes += info.Size()
	}
	return stats, nil
}

func (c *Cache) Purge() (int, error) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0, fmt.Errorf("cache: read dir: %w", err)
	}
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".json" && ext != ".tmp" {
			continue
		}
		if err := os.Remove(filepath.Join(c.dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return removed, fmt.Errorf("cache: remove %s: %w", entry.Name(), err)
		}
		removed++
	}
	return removed, nil
}

func (c *Cache) gc() {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	cutoff := c.nowFn().Add(-10 * c.ttl)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(c.dir, e.Name()))
		}
	}
}
