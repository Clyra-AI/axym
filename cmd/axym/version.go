package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newVersionCmd(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintln(stdout, version)
		},
	}
}
