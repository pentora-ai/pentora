package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Configuration management commands (CE version).`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello from CE config command!")
	},
}

func init() {
	// This will be added to root in root.go
}
