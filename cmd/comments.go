package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dsuranov/canvasprobe/internal/output"
)

var commentIDRe = regexp.MustCompile(`^[A-Za-z0-9:_-]+$`)

func newCommentsCmd() *cobra.Command {
	comments := &cobra.Command{
		Use:   "comments",
		Short: "Read or explicitly write file comments",
	}

	comments.AddCommand(&cobra.Command{
		Use:   "list <file-key>",
		Short: "List comments on a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFileKey(args[0]); err != nil {
				return err
			}
			client, err := newFigmaClient()
			if err != nil {
				return err
			}
			raw, err := client.GetComments(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.Write(cmd.OutOrStdout(), raw, output.FormatJSON, flags.compact)
		},
	})

	var message string
	var replyTo string
	create := &cobra.Command{
		Use:   "create <file-key>",
		Short: "Post a comment; never invoked by a read command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFileKey(args[0]); err != nil {
				return err
			}
			message = strings.TrimSpace(message)
			if message == "" {
				return fmt.Errorf("--message must not be empty")
			}
			if replyTo != "" && !commentIDRe.MatchString(replyTo) {
				return fmt.Errorf("invalid --reply-to comment ID")
			}
			client, err := newFigmaClient()
			if err != nil {
				return err
			}
			ctx, cancel := contextWithWriteTimeout(cmd)
			defer cancel()
			raw, err := client.PostComment(ctx, args[0], message, replyTo)
			if err != nil {
				return err
			}
			return output.Write(cmd.OutOrStdout(), raw, output.FormatJSON, flags.compact)
		},
	}
	create.Flags().StringVar(&message, "message", "", "comment text (required)")
	create.Flags().StringVar(&replyTo, "reply-to", "", "root comment ID to reply to")
	comments.AddCommand(create)

	var yes bool
	deleteCmd := &cobra.Command{
		Use:   "delete <file-key> <comment-id>",
		Short: "Delete a comment created by the authenticated user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFileKey(args[0]); err != nil {
				return err
			}
			if !commentIDRe.MatchString(args[1]) {
				return fmt.Errorf("invalid comment ID")
			}
			if !yes {
				return fmt.Errorf("refusing to delete without --yes")
			}
			client, err := newFigmaClient()
			if err != nil {
				return err
			}
			ctx, cancel := contextWithWriteTimeout(cmd)
			defer cancel()
			if err := client.DeleteComment(ctx, args[0], args[1]); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "{\"deleted\":true,\"comment_id\":%q}\n", args[1])
			return err
		},
	}
	deleteCmd.Flags().BoolVar(&yes, "yes", false, "confirm destructive deletion")
	comments.AddCommand(deleteCmd)

	return comments
}

func validateFileKey(fileKey string) error {
	if !fileKeyRe.MatchString(fileKey) {
		return fmt.Errorf("invalid file key %q: must match ^[A-Za-z0-9]+$", fileKey)
	}
	return nil
}

func contextWithWriteTimeout(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	timeout := time.Duration(cfg.Timeout) * time.Second
	return context.WithTimeout(cmd.Context(), timeout)
}
