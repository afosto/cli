package main

import (
	"github.com/afosto/cli/cmd/afosto/template"
	"github.com/afosto/cli/cmd/afosto/upload"
	"github.com/spf13/cobra"
	"os"
)

var (
	rootCmd = &cobra.Command{}
)

func init() {
	rootCmd.AddCommand(template.GetCommands()...)
	rootCmd.AddCommand(upload.GetCommands()...)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
