package plugin

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
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
			return executeInfoCommand(cmd, args[0], cacheDir)
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().String("output", "table", "Output format: json, table")
	cmd.Flags().Bool("quiet", false, "Suppress non-essential output")
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

// executeInfoCommand orchestrates the info command execution
func executeInfoCommand(cmd *cobra.Command, pluginName, cacheDir string) error {
	ctx := context.Background()

	// Setup dependencies
	formatter := getFormatter(cmd)
	svc, err := getPluginService(cacheDir)
	if err != nil {
		return err
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
}

// printPluginInfo formats and prints the plugin information
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
