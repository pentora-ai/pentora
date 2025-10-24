package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/rs/zerolog/log"
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

			// Create manifest manager
			manifestPath := filepath.Join(filepath.Dir(cacheDir), "registry.json")
			manifestMgr, err := plugin.NewManifestManager(manifestPath)
			if err != nil {
				return fmt.Errorf("create manifest manager: %w", err)
			}

			// Get plugins to verify
			var entries []*plugin.ManifestEntry
			if pluginName != "" {
				// Verify specific plugin
				entry, err := manifestMgr.Get(pluginName)
				if err != nil {
					return fmt.Errorf("plugin '%s' not found", pluginName)
				}
				entries = []*plugin.ManifestEntry{entry}
			} else {
				// Verify all plugins
				allEntries, err := manifestMgr.List()
				if err != nil {
					return fmt.Errorf("list plugins: %w", err)
				}
				entries = allEntries

				if len(entries) == 0 {
					fmt.Println("No plugins installed to verify.")
					return nil
				}
			}

			// Create verifier
			verifier := plugin.NewVerifier()

			// Verify each plugin
			fmt.Printf("Verifying %d plugin(s)...\n\n", len(entries))

			failedCount := 0
			for _, entry := range entries {
				pluginFile := filepath.Join(cacheDir, entry.ID, entry.Version, "plugin.yaml")

				// Check if file exists
				if _, err := os.Stat(pluginFile); os.IsNotExist(err) {
					fmt.Printf("✗ %s@%s - File not found: %s\n", entry.ID, entry.Version, pluginFile)
					failedCount++
					continue
				}

				// Verify checksum
				valid, err := verifier.VerifyFile(pluginFile, entry.Checksum)
				if err != nil {
					fmt.Printf("✗ %s@%s - Verification error: %v\n", entry.ID, entry.Version, err)
					log.Debug().Err(err).Str("plugin", entry.ID).Msg("Checksum verification failed")
					failedCount++
					continue
				}

				if valid {
					fmt.Printf("✓ %s@%s - OK\n", entry.ID, entry.Version)
				} else {
					fmt.Printf("✗ %s@%s - Checksum mismatch\n", entry.ID, entry.Version)
					failedCount++
				}
			}

			fmt.Println()
			if failedCount == 0 {
				fmt.Printf("All %d plugin(s) verified successfully.\n", len(entries))
				return nil
			}

			fmt.Printf("%d plugin(s) failed verification.\n", failedCount)
			return fmt.Errorf("verification failed for %d plugin(s)", failedCount)
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")
	cmd.Flags().StringVar(&pluginName, "plugin", "", "Verify specific plugin by name")

	return cmd
}
