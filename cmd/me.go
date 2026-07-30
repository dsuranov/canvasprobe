package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dsuranov/fgm-c/internal/output"
)

func newMeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show current user info",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Token == "" {
				return fmt.Errorf("no token: set FGM_C_TOKEN, FIGMA_API_TOKEN, use --token-stdin, or configure token")
			}

			client, err := newFigmaClient()
			if err != nil {
				return err
			}

			raw, err := client.GetMe(cmd.Context())
			if err != nil {
				return err
			}

			return output.Write(cmd.OutOrStdout(), raw, output.FormatJSON, flags.compact)
		},
	}
}
