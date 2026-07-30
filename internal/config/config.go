package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Token         string `yaml:"token"`
	DefaultFormat string `yaml:"default_format"`
	CacheTTL      int    `yaml:"cache_ttl"`
	Timeout       int    `yaml:"timeout"`
}

func Load() (*Config, error) {
	cfg := &Config{
		DefaultFormat: "json",
		CacheTTL:      300,
		Timeout:       30,
	}

	if err := loadYAML(cfg); err != nil {
		return nil, err
	}

	loadEnv(cfg)

	switch cfg.DefaultFormat {
	case "json", "table", "csv":
	default:
		return nil, fmt.Errorf("invalid default_format %q: expected json, table, or csv", cfg.DefaultFormat)
	}
	if cfg.CacheTTL < 0 {
		return nil, fmt.Errorf("cache_ttl must be >= 0")
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("timeout must be > 0")
	}

	return cfg, nil
}

func loadYAML(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat config: %w", err)
	}
	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("config %s must not be accessible by group or others; run chmod 600 %s", path, path)
	}

	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	return nil
}

func loadEnv(cfg *Config) {
	if v := os.Getenv("FIGMA_API_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("FGM_C_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("FGM_C_DEFAULT_FORMAT"); v != "" {
		cfg.DefaultFormat = v
	}
	if v := os.Getenv("FGM_C_CACHE_TTL"); v != "" {
		cfg.CacheTTL = parseEnvInt("FGM_C_CACHE_TTL", v, cfg.CacheTTL)
	}
	if v := os.Getenv("FGM_C_TIMEOUT"); v != "" {
		cfg.Timeout = parseEnvInt("FGM_C_TIMEOUT", v, cfg.Timeout)
	}
}

func parseEnvInt(name, value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: ignoring invalid %s=%q\n", name, value)
		return fallback
	}
	return n
}

func configPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "fgm-c", "config.yaml"), nil
}
