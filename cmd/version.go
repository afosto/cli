package cmd

import (
	"github.com/spf13/cobra"
	"log"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Connect with Afosto/IO",
		Long:  `Connect with Afosto`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Print("Afosto/IO CLI")
		}})

}
