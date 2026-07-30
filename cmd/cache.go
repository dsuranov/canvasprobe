package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/dsuranov/fgm-c/internal/figma"
)

func newCacheCmd() *cobra.Command {
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Inspect or purge the local response cache",
	}

	cacheCmd.AddCommand(&cobra.Command{
		Use:   "dir",
		Short: "Print the cache directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), figma.DefaultCacheDir())
			return err
		},
	})

	cacheCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show entry count and size",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cache, err := figma.NewCache(figma.DefaultCacheDir(), time.Duration(cfg.CacheTTL)*time.Second)
			if err != nil {
				return err
			}
			stats, err := cache.Status()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "entries=%d bytes=%d\n", stats.Entries, stats.Bytes)
			return err
		},
	})

	cacheCmd.AddCommand(&cobra.Command{
		Use:   "purge",
		Short: "Delete all FGM-C cache entries",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cache, err := figma.NewCache(figma.DefaultCacheDir(), time.Minute)
			if err != nil {
				return err
			}
			removed, err := cache.Purge()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "removed=%d\n", removed)
			return err
		},
	})

	return cacheCmd
}
