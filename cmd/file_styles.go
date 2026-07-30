package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dsuranov/fgm-c/internal/output"
)

func runFileStyles(cmd *cobra.Command, fileKey string) error {
	if fFlags.depth != -1 || fFlags.ids != "" || fFlags.version != "" {
		return fmt.Errorf("styles does not accept --depth, --ids, or --version")
	}
	client, err := newFigmaClient()
	if err != nil {
		return err
	}

	raw, err := client.GetFileStyles(cmd.Context(), fileKey)
	if err != nil {
		return err
	}

	// Extract the styles array from the Figma API envelope.
	arr, err := applyPipeline(cmd.Context(), raw, "", ".meta.styles // []")
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
