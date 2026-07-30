package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dsuranov/fgm-c/internal/jsonpipe"
	"github.com/dsuranov/fgm-c/internal/output"
)

var fileKeyRe = regexp.MustCompile(`^[A-Za-z0-9]+$`)
var nodeIDRe = regexp.MustCompile(`^\d+:\d+$`)
var nodeIDsRe = regexp.MustCompile(`^\d+:\d+(,\d+:\d+)*$`)

type fileFlags struct {
	depth   int
	ids     string
	version string
	fields  string
	jq      string
}

var fFlags fileFlags

func newFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file <key> [nodes <ids> | components | styles]",
		Short: "Fetch design file data",
		Long: `Fetch design file data from the Figma REST API.

Operations (specified after the file key):
  fgm-c file <key>                  Fetch file tree (default: --depth 2)
  fgm-c file <key> nodes <ids>      Fetch specific nodes by comma-separated ID
  fgm-c file <key> components       List file components
  fgm-c file <key> styles           List file styles`,
		Args: cobra.MinimumNArgs(1),
		RunE: fileRunE,
	}

	cmd.Flags().IntVar(&fFlags.depth, "depth", -1, "tree depth limit; 0=no limit, -1=auto (default 2 for file fetch)")
	cmd.Flags().StringVar(&fFlags.ids, "ids", "", "comma-separated node IDs for server-side filtering")
	cmd.Flags().StringVar(&fFlags.version, "version", "", "file version ID")
	cmd.Flags().StringVar(&fFlags.fields, "fields", "", "comma-separated fields for components/styles output")
	cmd.Flags().StringVar(&fFlags.jq, "jq", "", "jq expression to apply to the response")

	return cmd
}

func fileRunE(cmd *cobra.Command, args []string) error {
	fileKey := args[0]
	if !fileKeyRe.MatchString(fileKey) {
		return fmt.Errorf("invalid file key %q: must match ^[A-Za-z0-9]+$", fileKey)
	}

	if len(args) > 1 {
		switch args[1] {
		case "nodes":
			if len(args) != 3 {
				return fmt.Errorf("nodes: missing <ids> argument")
			}
			return runFileNodes(cmd, fileKey, args[2])
		case "components":
			if len(args) != 2 {
				return fmt.Errorf("components: unexpected arguments")
			}
			return runFileComponents(cmd, fileKey)
		case "styles":
			if len(args) != 2 {
				return fmt.Errorf("styles: unexpected arguments")
			}
			return runFileStyles(cmd, fileKey)
		default:
			return fmt.Errorf("unknown operation %q; valid: nodes, components, styles", args[1])
		}
	}
	if len(args) != 1 {
		return fmt.Errorf("unexpected arguments")
	}

	return runFile(cmd, fileKey)
}

func runFile(cmd *cobra.Command, fileKey string) error {
	if fFlags.fields != "" {
		return fmt.Errorf("--fields is supported only for components and styles; use --jq for file trees")
	}
	if fFlags.depth < -1 {
		return fmt.Errorf("--depth must be -1, 0, or a positive integer")
	}

	client, err := newFigmaClient()
	if err != nil {
		return err
	}

	depth := fFlags.depth
	serverDepth := 0

	switch {
	case depth == -1 && fFlags.ids == "":
		serverDepth = 2
		fmt.Fprintln(cmd.ErrOrStderr(), "warn: applied default --depth 2; pass --depth 0 to disable, --ids to fetch specific nodes")
	case depth == 0:
		serverDepth = 0
		fmt.Fprintln(cmd.ErrOrStderr(), "warn: --depth 0 disables depth limit; response can exceed 10 MB")
	case depth == -1 && fFlags.ids != "":
		serverDepth = 0
	default:
		serverDepth = depth
	}

	var ids []string
	if fFlags.ids != "" {
		ids = strings.Split(fFlags.ids, ",")
		for _, id := range ids {
			if !nodeIDRe.MatchString(strings.TrimSpace(id)) {
				return fmt.Errorf("invalid node ID %q: must match \\d+:\\d+", id)
			}
		}
	}

	raw, err := client.GetFile(cmd.Context(), fileKey, serverDepth, ids, fFlags.version)
	if err != nil {
		return err
	}

	data, err := applyPipeline(cmd.Context(), raw, "", fFlags.jq)
	if err != nil {
		return err
	}

	return output.Write(cmd.OutOrStdout(), data, resolveFormat(cfg.DefaultFormat), flags.compact)
}

// applyPipeline runs ApplyFields → ApplyJQ in order.
// Empty fieldsStr skips ApplyFields; empty jqExpr skips ApplyJQ.
func applyPipeline(ctx context.Context, raw []byte, fieldsStr, jqExpr string) (json.RawMessage, error) {
	data := json.RawMessage(raw)

	if fieldsStr != "" {
		fields := strings.Split(fieldsStr, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
			if fields[i] == "" {
				return nil, fmt.Errorf("apply fields: field names must not be empty")
			}
		}
		var err error
		data, err = jsonpipe.ApplyFields(data, fields)
		if err != nil {
			return nil, fmt.Errorf("apply fields: %w", err)
		}
	}

	if jqExpr != "" {
		jqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		var err error
		data, err = jsonpipe.ApplyJQ(jqCtx, data, jqExpr)
		if err != nil {
			return nil, fmt.Errorf("apply jq: %w", err)
		}
	}

	return data, nil
}

func resolveFormat(defaultFmt string) output.Format {
	switch defaultFmt {
	case "table":
		return output.FormatTable
	case "csv":
		return output.FormatCSV
	default:
		return output.FormatJSON
	}
}

func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		fi, err := f.Stat()
		return err == nil && fi.Mode()&os.ModeCharDevice != 0
	}
	return false
}
