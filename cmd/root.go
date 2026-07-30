package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dsuranov/fgm-c/internal/config"
)

var version = "dev"

type rootFlags struct {
	tokenStdin bool
	format     string
	noCache    bool
	verbose    bool
	compact    bool
}

var flags rootFlags
var cfg *config.Config

func newRootCmd() *cobra.Command {
	flags = rootFlags{}
	fFlags = fileFlags{}

	root := &cobra.Command{
		Use:           "fgm-c",
		Short:         "Inspect design data and post explicit review comments",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = config.Load()
			if err != nil {
				return err
			}

			if flags.tokenStdin {
				token, readErr := readToken(cmd.InOrStdin())
				if readErr != nil {
					return readErr
				}
				cfg.Token = token
			}
			if flags.format != "" {
				switch flags.format {
				case "json", "table", "csv":
				default:
					return fmt.Errorf("invalid --format %q: expected json, table, or csv", flags.format)
				}
				cfg.DefaultFormat = flags.format
			}
			return nil
		},
	}

	root.PersistentFlags().BoolVar(&flags.tokenStdin, "token-stdin", false, "read API token from stdin")
	root.PersistentFlags().StringVar(&flags.format, "format", "", "output format: json, table, csv")
	root.PersistentFlags().BoolVar(&flags.noCache, "no-cache", false, "disable cache read and write")
	root.PersistentFlags().BoolVar(&flags.verbose, "verbose", false, "print request details to stderr")
	root.PersistentFlags().BoolVar(&flags.compact, "compact", false, "compact JSON output (no indentation)")

	root.AddCommand(newMeCmd())
	root.AddCommand(newFileCmd())
	root.AddCommand(newCommentsCmd())
	root.AddCommand(newCacheCmd())

	return root
}

func readToken(r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, 16*1024))
	if err != nil {
		return "", fmt.Errorf("read token from stdin: %w", err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("token from stdin is empty")
	}
	if strings.ContainsAny(token, " \t\r\n") {
		return "", fmt.Errorf("token from stdin contains whitespace")
	}
	return token, nil
}

func Execute() int {
	root := newRootCmd()

	var usageErr bool
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		usageErr = true
		return err
	})

	if err := root.ExecuteContext(context.Background()); err != nil {
		printErr(err)
		if usageErr {
			return 2
		}
		return exitCode(err)
	}
	return 0
}

func printErr(err error) {
	if err != nil {
		// cobra already prints usage for bad-args errors, we print the rest
		os.Stderr.WriteString("error: " + err.Error() + "\n")
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if isNetworkError(err) {
		return 3
	}
	return 1
}

func isNetworkError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}
