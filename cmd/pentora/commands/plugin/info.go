package plugin

import (
	"context"
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
			ctx := context.Background()

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

			// Call service layer
			info, err := svc.GetInfo(ctx, pluginName)
			if err != nil {
				if err == plugin.ErrPluginNotFound {
					return fmt.Errorf("plugin '%s' not found\n\nUse 'pentora plugin list' to see installed plugins", pluginName)
				}
				return fmt.Errorf("get plugin info: %w", err)
			}

			// Print results
			printPluginInfo(info)

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")

	return cmd
}

// printPluginInfo formats and prints the plugin information
func printPluginInfo(info *plugin.PluginInfo) {
	fmt.Printf("Plugin: %s\n", info.Name)
	fmt.Printf("Version: %s\n", info.Version)
	fmt.Printf("Checksum: %s\n", info.Checksum)
	fmt.Printf("Download URL: %s\n", info.DownloadURL)
	fmt.Printf("Installed: %s\n", info.InstalledAt.Format("2006-01-02 15:04:05"))

	if info.CacheDir != "" {
		fmt.Printf("Location: %s\n", info.CacheDir)
		if info.CacheSize > 0 {
			fmt.Printf("Size: %s\n", formatBytes(info.CacheSize))
		}
	}
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
