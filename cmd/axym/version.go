package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newVersionCmd(stdout io.Writer, global *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if global.JSON {
				return printJSON(stdout, envelope{
					OK:      true,
					Command: "version",
					Data:    map[string]any{"version": version},
				})
			}
			if !global.Quiet {
				_, _ = fmt.Fprintln(stdout, version)
			}
			return nil
		},
	}
}
