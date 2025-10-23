package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/spf13/cobra"
)

func newInfoCommand() *cobra.Command {
	var cacheDir string

	cmd := &cobra.Command{
		Use:   "info <plugin-name>",
		Short: "Show detailed information about a plugin",
		Long: `Display detailed information about an installed plugin.

Shows plugin metadata including name, version, checksum, download URL,
installation path, and cache size.`,
		Example: `  # Show info for a specific plugin
  pentora plugin info ssh-cve-2024-6387

  # Use custom cache directory
  pentora plugin info ssh-cve-2024-6387 --cache-dir /custom/path`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]

			// Use default cache dir if not specified
			if cacheDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("get home directory: %w", err)
				}
				cacheDir = filepath.Join(homeDir, ".pentora", "plugins", "cache")
			}

			// Create manifest manager to find plugin
			manifestPath := filepath.Join(filepath.Dir(cacheDir), "registry.json")
			manifestMgr, err := plugin.NewManifestManager(manifestPath)
			if err != nil {
				return fmt.Errorf("create manifest manager: %w", err)
			}

			// Get plugin from manifest
			entry, err := manifestMgr.Get(pluginName)
			if err != nil {
				// Simple string check for "not found" error
				return fmt.Errorf("plugin '%s' not found\n\nUse 'pentora plugin list' to see installed plugins", pluginName)
			}

			// Display plugin info
			fmt.Printf("Plugin: %s\n", entry.Name)
			fmt.Printf("Version: %s\n", entry.Version)
			fmt.Printf("Checksum: %s\n", entry.Checksum)
			fmt.Printf("Download URL: %s\n", entry.DownloadURL)
			fmt.Printf("Installed: %s\n", entry.InstalledAt.Format("2006-01-02 15:04:05"))

			// Get cache directory info
			pluginDir := filepath.Join(cacheDir, entry.Name, entry.Version)
			if _, err := os.Stat(pluginDir); err == nil {
				fmt.Printf("Location: %s\n", pluginDir)

				// Calculate directory size
				var totalSize int64
				err := filepath.Walk(pluginDir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() {
						totalSize += info.Size()
					}
					return nil
				})
				if err == nil {
					fmt.Printf("Size: %s\n", formatBytes(totalSize))
				}
			} else {
				fmt.Printf("Location: %s (not found)\n", pluginDir)
			}

			// Show registry URL if available
			registryURL, err := manifestMgr.GetRegistryURL()
			if err == nil && registryURL != "" {
				fmt.Printf("\nRegistry: %s\n", registryURL)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")

	return cmd
}

// formatBytes formats bytes as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
