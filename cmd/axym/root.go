package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newRootCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	flags := &globalFlags{}

	root := &cobra.Command{
		Use:   "axym",
		Short: "Axym deterministic evidence CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.JSON {
				return printJSON(stdout, envelope{
					OK:      true,
					Command: "root",
					Data:    map[string]any{"name": "axym", "version": version},
				})
			}
			if !flags.Quiet {
				_, _ = fmt.Fprintln(stdout, "axym")
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVar(&flags.JSON, "json", false, "Emit machine-readable JSON output")
	root.PersistentFlags().BoolVar(&flags.Quiet, "quiet", false, "Suppress human-readable output")
	root.PersistentFlags().BoolVar(&flags.Explain, "explain", false, "Emit additional rationale in human output")

	root.AddCommand(newVersionCmd(stdout, flags))
	root.AddCommand(newCollectCmd(stdout, flags))
	root.AddCommand(newVerifyCmd(stdout, stderr, flags))
	return root
}

func execute(args []string, stdout io.Writer, stderr io.Writer) int {
	root := newRootCmd(stdout, stderr)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		if codeErr, ok := err.(interface{ ExitCode() int }); ok {
			return codeErr.ExitCode()
		}
		_, _ = fmt.Fprintln(stderr, err.Error())
		return exitRuntimeFailure
	}
	return exitSuccess
}
