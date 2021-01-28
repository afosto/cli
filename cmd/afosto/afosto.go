package main

import (
	"github.com/afosto/cli/cmd/afosto/auth"
	"github.com/afosto/cli/cmd/afosto/common"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	Execute()
}

var (
	rootCmd = &cobra.Command{}
)

func init() {
	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "auth",
			Short: "Connect with Afosto/IO",
			Long:  `Connect with Afosto`,
			Run: func(cmd *cobra.Command, args []string) {
				auth.Login()
			}},
		&cobra.Command{
			Use:   "version",
			Short: "Show version",
			Long:  `Show the tool version`,
			Run: func(cmd *cobra.Command, args []string) {
				common.Version()
			}})

}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
