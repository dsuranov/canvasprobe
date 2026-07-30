package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/dsuranov/fgm-c/internal/output"
)

func runFileComponents(cmd *cobra.Command, fileKey string) error {
	if fFlags.depth != -1 || fFlags.ids != "" || fFlags.version != "" {
		return fmt.Errorf("components does not accept --depth, --ids, or --version")
	}
	client, err := newFigmaClient()
	if err != nil {
		return err
	}

	raw, err := client.GetFileComponents(cmd.Context(), fileKey)
	if err != nil {
		return err
	}

	// Extract the components array from the Figma API envelope.
	arr, err := applyPipeline(cmd.Context(), raw, "", ".meta.components // []")
	if err != nil {
		return err
	}

	// Apply user-supplied filters on the extracted array.
	data, err := applyPipeline(cmd.Context(), arr, fFlags.fields, fFlags.jq)
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	return output.Write(w, data, autoFormat(w), flags.compact)
}

// autoFormat returns table when stdout is a terminal, json otherwise.
// Respects an explicit --format CLI flag or non-default config/env format.
func autoFormat(w io.Writer) output.Format {
	if flags.format != "" || cfg.DefaultFormat != "json" {
		return resolveFormat(cfg.DefaultFormat)
	}
	if isTerminal(w) {
		return output.FormatTable
	}
	return output.FormatJSON
}
