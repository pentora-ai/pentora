package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
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
  pentora plugin info ssh-cve-2024-6387 --cache-dir /custom/path

  # JSON output
  pentora plugin info ssh-cve-2024-6387 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]
			ctx := context.Background()

			// Create formatter
			outputMode := format.ParseMode(cmd.Flag("output").Value.String())
			quiet, _ := cmd.Flags().GetBool("quiet")
			noColor, _ := cmd.Flags().GetBool("no-color")
			formatter := format.New(os.Stdout, os.Stderr, outputMode, quiet, !noColor)

			// Use default cache dir if not specified
			if cacheDir == "" {
				storageConfig, err := storage.DefaultConfig()
				if err != nil {
					return fmt.Errorf("get storage config: %w", err)
				}
				cacheDir = filepath.Join(storageConfig.WorkspaceRoot, "plugins", "cache")
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
					return formatter.PrintError(fmt.Errorf("plugin '%s' not found\n\nUse 'pentora plugin list' to see installed plugins", pluginName))
				}
				return formatter.PrintError(err)
			}

			// Print results
			return printPluginInfo(formatter, info)
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().String("output", "table", "Output format: json, table")
	cmd.Flags().Bool("quiet", false, "Suppress non-essential output")
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

// printPluginInfo formats and prints the plugin information using the formatter
func printPluginInfo(f format.Formatter, info *plugin.PluginInfo) error {
	// Build key-value table
	rows := [][]string{
		{"Name", info.Name},
		{"Version", info.Version},
		{"Checksum", info.Checksum},
		{"Download URL", info.DownloadURL},
		{"Installed", info.InstalledAt.Format("2006-01-02 15:04:05")},
	}

	if info.CacheDir != "" {
		rows = append(rows, []string{"Location", info.CacheDir})
		if info.CacheSize > 0 {
			rows = append(rows, []string{"Size", formatBytes(info.CacheSize)})
		}
	}

	return f.PrintTable([]string{"Property", "Value"}, rows)
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
