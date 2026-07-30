package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dsuranov/canvasprobe/internal/output"
)

func runFileNodes(cmd *cobra.Command, fileKey, idsStr string) error {
	if fFlags.fields != "" {
		return fmt.Errorf("--fields is supported only for components and styles; use --jq for node responses")
	}
	if fFlags.ids != "" || fFlags.version != "" {
		return fmt.Errorf("nodes does not accept --ids or --version")
	}
	if fFlags.depth < -1 {
		return fmt.Errorf("--depth must be -1, 0, or a positive integer")
	}
	if !nodeIDsRe.MatchString(idsStr) {
		return fmt.Errorf("invalid node IDs %q: must match \\d+:\\d+(,\\d+:\\d+)*", idsStr)
	}

	ids := strings.Split(idsStr, ",")

	client, err := newFigmaClient()
	if err != nil {
		return err
	}

	serverDepth := 0
	if fFlags.depth > 0 {
		serverDepth = fFlags.depth
	}

	raw, err := client.GetFileNodes(cmd.Context(), fileKey, ids, serverDepth)
	if err != nil {
		return err
	}

	data, err := applyPipeline(cmd.Context(), raw, "", fFlags.jq)
	if err != nil {
		return err
	}

	return output.Write(cmd.OutOrStdout(), data, resolveFormat(cfg.DefaultFormat), flags.compact)
}
