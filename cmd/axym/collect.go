package main

import (
	"io"
	"time"

	"github.com/spf13/cobra"
)

func newCollectCmd(stdout io.Writer, global *globalFlags) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect evidence",
		RunE: func(cmd *cobra.Command, args []string) error {
			result := map[string]any{
				"status":    "ready",
				"dry_run":   dryRun,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "collect", Data: result})
			}
			if !global.Quiet {
				if dryRun {
					printText(stdout, "collect dry-run ready", global.Quiet)
				} else {
					printText(stdout, "collect ready", global.Quiet)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate collection configuration without writing artifacts")
	return cmd
}
