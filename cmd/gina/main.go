// Package main is sample code for github.com/nao1215/imaging
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// main is entry point of gina command.
func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newRootCmd return root command of gina. This command has subcommands.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "gina",
		Long: `gina --Go Image 'N' Assistance-- is simple image processing CLI tool.

The gina was created to help developers understand 'how to use the image
processing methods provided by the nao1215/imaging package'.`,
	}
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newBugReportCmd())
	cmd.AddCommand(newResizeCmd())
	cmd.AddCommand(newSharpenCmd())
	cmd.AddCommand(newBlurCmd())
	cmd.AddCommand(newContrastCmd())
	cmd.AddCommand(newGammaCmd())
	return cmd
}
