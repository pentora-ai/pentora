// cmd/pentora/pentora.go
package main

import (
	"fmt"
	"os"

	"github.com/pentoraai/pentora/cmd/license"
	"github.com/pentoraai/pentora/cmd/version"
	"github.com/pentoraai/pentora/pkg/cli"
	lic "github.com/pentoraai/pentora/pkg/license"

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

func loadGlobalLicense() {
	lic.GlobalStatus = lic.Check(lic.GetDefaultLicensePath(), lic.GetPublicKeyPath())

	if lic.GlobalStatus.Valid {
		fmt.Println("🔐 License OK –", lic.GlobalStatus.Payload.Licensee)
	} else if lic.GlobalStatus.Error != nil {
		fmt.Println("⚠️ License error:", lic.GlobalStatus.Error)
	} else {
		fmt.Println("⚠️ No license found. Running in free mode.")
	}
}

func init() {
	rootCmd.AddCommand(cli.ServeCmd)
	rootCmd.AddCommand(cli.ScanCmd)
	rootCmd.AddCommand(license.LicenseCmd)
	rootCmd.AddCommand(version.VersionCmd)
}

func main() {
	loadGlobalLicense()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
