package cmd

import (
	"os"
	"time"

	"github.com/dsuranov/canvasprobe/internal/figma"
)

var clientOptionsOverride func(*figma.Options)

func newFigmaClient() (*figma.Client, error) {
	var cache *figma.Cache
	if !flags.noCache && cfg.CacheTTL > 0 {
		var err error
		cache, err = figma.NewCache(figma.DefaultCacheDir(), time.Duration(cfg.CacheTTL)*time.Second)
		if err != nil {
			return nil, err
		}
	}
	opts := figma.Options{
		Token:     cfg.Token,
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
		UserAgent: "canvasprobe/" + version,
		Verbose:   flags.verbose,
		Stderr:    os.Stderr,
		Cache:     cache,
	}
	if clientOptionsOverride != nil {
		clientOptionsOverride(&opts)
	}
	return figma.New(opts)
}
