package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/spf13/cobra"
)

func newVerifyCommand() *cobra.Command {
	var (
		cacheDir   string
		pluginName string
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify plugin checksums",
		Long: `Verify the integrity of installed plugins by checking their SHA-256 checksums.

By default, verifies all installed plugins. Use --plugin to verify a specific plugin.

Exit codes:
  0 - All plugins verified successfully
  1 - One or more plugins failed verification or error occurred`,
		Example: `  # Verify all installed plugins
  pentora plugin verify

  # Verify a specific plugin
  pentora plugin verify --plugin ssh-cve-2024-6387

  # Verify plugins in custom cache directory
  pentora plugin verify --cache-dir /custom/path`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use default cache dir if not specified
			if cacheDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("get home directory: %w", err)
				}
				cacheDir = filepath.Join(homeDir, ".pentora", "plugins", "cache")
			}

			// Create service
			svc, err := plugin.NewService(cacheDir)
			if err != nil {
				return fmt.Errorf("create plugin service: %w", err)
			}

			// Build verify options
			opts := plugin.VerifyOptions{
				PluginID: pluginName,
			}

			// Call service layer
			result, err := svc.Verify(cmd.Context(), opts)
			if err != nil {
				return err
			}

			// Print results
			printVerifyResult(result)

			// Return error if any plugins failed
			if result.FailedCount > 0 {
				return fmt.Errorf("verification failed for %d plugin(s)", result.FailedCount)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")
	cmd.Flags().StringVar(&pluginName, "plugin", "", "Verify specific plugin by name")

	return cmd
}

// printVerifyResult formats and prints the verify result
func printVerifyResult(result *plugin.VerifyResult) {
	if result.TotalCount == 0 {
		fmt.Println("No plugins installed to verify.")
		return
	}

	fmt.Printf("Verifying %d plugin(s)...\n\n", result.TotalCount)

	// Print individual results
	for _, r := range result.Results {
		if r.Valid {
			fmt.Printf("✓ %s@%s - OK\n", r.ID, r.Version)
		} else {
			switch r.ErrorType {
			case "missing":
				fmt.Printf("✗ %s@%s - File not found\n", r.ID, r.Version)
			case "checksum":
				fmt.Printf("✗ %s@%s - Checksum mismatch\n", r.ID, r.Version)
			case "error":
				fmt.Printf("✗ %s@%s - Verification error: %v\n", r.ID, r.Version, r.Error)
			default:
				fmt.Printf("✗ %s@%s - Failed\n", r.ID, r.Version)
			}
		}
	}

	// Print summary
	fmt.Println()
	if result.FailedCount == 0 {
		fmt.Printf("All %d plugin(s) verified successfully.\n", result.TotalCount)
	} else {
		fmt.Printf("%d plugin(s) failed verification.\n", result.FailedCount)
	}
}
