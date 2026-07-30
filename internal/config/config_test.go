package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultFormat != "json" || cfg.CacheTTL != 300 || cfg.Timeout != 30 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestEnvOverridesDefault(t *testing.T) {
	clearEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FIGMA_API_TOKEN", "upstream-token")
	t.Setenv("FGM_C_TOKEN", "primary-token")
	t.Setenv("FGM_C_DEFAULT_FORMAT", "table")
	t.Setenv("FGM_C_CACHE_TTL", "60")
	t.Setenv("FGM_C_TIMEOUT", "45")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "primary-token" {
		t.Errorf("Token: got %q", cfg.Token)
	}
	if cfg.DefaultFormat != "table" || cfg.CacheTTL != 60 || cfg.Timeout != 45 {
		t.Fatalf("unexpected env config: %+v", cfg)
	}
}

func TestYAMLAndEnvPrecedence(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "fgm-c")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	content := "token: yaml-token\ndefault_format: csv\ncache_ttl: 600\ntimeout: 60\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FGM_C_TOKEN", "env-token")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "env-token" || cfg.DefaultFormat != "csv" || cfg.CacheTTL != 600 || cfg.Timeout != 60 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestUnsafeConfigPermissionsFail(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := filepath.Join(dir, "fgm-c")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(path, []byte("token: secret\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected insecure-permissions error")
	}
}

func TestInvalidValuesFail(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"format", "FGM_C_DEFAULT_FORMAT", "xml"},
		{"negative ttl", "FGM_C_CACHE_TTL", "-1"},
		{"zero timeout", "FGM_C_TIMEOUT", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			t.Setenv(tt.key, tt.value)
			if _, err := Load(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"FIGMA_API_TOKEN",
		"FGM_C_TOKEN",
		"FGM_C_DEFAULT_FORMAT",
		"FGM_C_CACHE_TTL",
		"FGM_C_TIMEOUT",
		"XDG_CONFIG_HOME",
	} {
		t.Setenv(key, "")
	}
}
