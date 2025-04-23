// cmd/pentora/main.go
package main

import (
	"fmt"
	"os"

	"github.com/pentoraai/pentora/pkg/cli"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pentora",
	Short: "Pentora - Platform-independent vulnerability scanner",
	Long:  `Pentora is a cross-platform security scanner designed to find vulnerabilities and misconfigurations in your infrastructure.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Pentora CLI. Use --help for available commands.")
	},
}

func init() {
	rootCmd.AddCommand(cli.ServeCmd)
	rootCmd.AddCommand(cli.ScanCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
