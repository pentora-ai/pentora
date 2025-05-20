package license

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pentora-ai/pentora/pkg/license"
	"github.com/spf13/cobra"
)

var licenseFile string

// Root license command
var LicenseCmd = &cobra.Command{
	Use:   "license",
	Short: "License management commands",
}

// Subcommand: license install
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a new Pentora license file",
	Run: func(cmd *cobra.Command, args []string) {
		if licenseFile == "" {
			fmt.Println("❌ Please provide a license file with --file")
			os.Exit(1)
		}

		content, err := os.ReadFile(licenseFile)
		if err != nil {
			fmt.Printf("❌ Failed to read license file: %v\n", err)
			os.Exit(1)
		}

		destPath := license.GetDefaultLicensePath()
		destDir := filepath.Dir(destPath)

		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Printf("❌ Failed to create license directory: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			fmt.Printf("❌ Failed to write license file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ License file installed to %s\n", destPath)

		status := license.Check(destPath, license.GetPublicKeyPath())
		if status.Valid {
			fmt.Printf("🔐 License valid – %s (%s)\n", status.Payload.Licensee, status.Payload.LicenseType)
		} else {
			fmt.Printf("⚠️ License saved but invalid: %v\n", status.Error)
		}
	},
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current license information",
	Run: func(cmd *cobra.Command, args []string) {
		path := license.GetDefaultLicensePath()
		pub := license.GetPublicKeyPath()

		status := license.Check(path, pub)

		fmt.Println("🔍 License File:", path)

		if status.Valid {
			fmt.Println("✅ License is valid.")
			fmt.Println("👤 Licensee:", status.Payload.Licensee)
			fmt.Println("🏢 Organization:", status.Payload.Organization)
			fmt.Println("📅 Issued At:", status.Payload.IssuedAt)
			fmt.Println("📅 Expires At:", status.Payload.ExpiresAt)
			fmt.Println("📦 Type:", status.Payload.LicenseType)
			fmt.Println("🧩 Features:")
			for _, f := range status.Payload.Features {
				fmt.Println("  -", f)
			}
		} else {
			if status.Error != nil {
				fmt.Println("❌ License invalid:", status.Error)
			} else {
				fmt.Println("⚠️ No license found. Running in free mode.")
				fmt.Println("🧩 Available Free Features:")
				for _, f := range license.GetFreeFeatures() {
					fmt.Println("  -", f)
				}
			}
		}
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the installed license file",
	Run: func(cmd *cobra.Command, args []string) {
		path := license.GetDefaultLicensePath()
		pub := license.GetPublicKeyPath()

		fmt.Println("🔍 Validating license at:", path)
		status := license.Check(path, pub)

		switch {
		case status.Valid:
			fmt.Println("✅ License is valid.")
			fmt.Println("👤 Licensee:", status.Payload.Licensee)
			fmt.Println("📅 Expires At:", status.Payload.ExpiresAt)

		case status.Error != nil:
			fmt.Println("❌ License validation failed:")
			fmt.Println("   →", status.Error)

		default:
			fmt.Println("⚠️ No license file found.")
		}
	},
}

func init() {
	installCmd.Flags().StringVarP(&licenseFile, "file", "f", "", "Path to the license file")
	LicenseCmd.AddCommand(installCmd)
	LicenseCmd.AddCommand(showCmd)
	LicenseCmd.AddCommand(validateCmd)
}
