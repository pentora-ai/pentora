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

func newListCommand() *cobra.Command {
	var (
		cacheDir string
		verbose  bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all installed plugins",
		Long: `List all plugins currently installed in the cache.

Displays plugin name, version, and installation status for each plugin
found in the plugin cache directory.`,
		Example: `  # List all installed plugins
  pentora plugin list

  # List plugins with detailed information
  pentora plugin list --verbose

  # List plugins from custom cache directory
  pentora plugin list --cache-dir /custom/path

  # JSON output
  pentora plugin list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			plugins, err := svc.List(ctx)
			if err != nil {
				return formatter.PrintError(err)
			}

			// Print results
			return printListResult(formatter, plugins, verbose)
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed information")
	cmd.Flags().String("output", "table", "Output format: json, table")
	cmd.Flags().Bool("quiet", false, "Suppress non-essential output")
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

// printListResult formats and prints the list result using the formatter
func printListResult(f format.Formatter, plugins []*plugin.PluginInfo, verbose bool) error {
	// JSON mode: output complete structured result
	if f.IsJSON() {
		result := map[string]any{
			"plugins": plugins,
			"count":   len(plugins),
		}
		return f.PrintJSON(result)
	}

	// Table mode: use existing table + summary pattern
	if len(plugins) == 0 {
		if err := f.PrintSummary("No plugins installed."); err != nil {
			return err
		}
		if err := f.PrintSummary(""); err != nil {
			return err
		}
		if err := f.PrintSummary("To install plugins, use:"); err != nil {
			return err
		}
		return f.PrintSummary("  pentora plugin install <plugin-name>")
	}

	// Build table headers and rows
	var headers []string
	var rows [][]string

	if verbose {
		headers = []string{"Name", "Version", "Checksum", "Download URL"}
		for _, p := range plugins {
			rows = append(rows, []string{
				p.Name,
				p.Version,
				truncateChecksum(p.Checksum),
				truncateURL(p.DownloadURL),
			})
		}
	} else {
		headers = []string{"Name", "Version"}
		for _, p := range plugins {
			rows = append(rows, []string{p.ID, p.Version})
		}
	}

	// Print table
	if err := f.PrintTable(headers, rows); err != nil {
		return err
	}

	// Print summary
	return f.PrintSummary(fmt.Sprintf("Found %d plugin(s)", len(plugins)))
}

// truncateChecksum truncates checksum for display (shows first 12 chars)
func truncateChecksum(checksum string) string {
	if len(checksum) > 12 {
		return checksum[:12] + "..."
	}
	return checksum
}

// truncateURL truncates URL for display (shows last 40 chars)
func truncateURL(url string) string {
	if len(url) > 40 {
		return "..." + url[len(url)-37:]
	}
	return url
}
