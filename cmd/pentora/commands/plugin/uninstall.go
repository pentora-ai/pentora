package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
)

func newUninstallCommand() *cobra.Command {
	var cacheDir string

	cmd := &cobra.Command{
		Use:     "uninstall <plugin-name>",
		Aliases: []string{"remove", "rm"},
		Short:   "Uninstall plugins",
		Long: `Uninstall (remove) plugins from the local cache.

This command removes plugins from the cache directory. You can uninstall specific plugins by name,
all plugins in a category, or all plugins at once.`,
		Example: `  # Uninstall specific plugin
  pentora plugin uninstall ssh-cve-2024-6387

  # Uninstall all SSH plugins
  pentora plugin uninstall --category ssh

  # Uninstall all plugins
  pentora plugin uninstall --all

  # Alternative commands (aliases)
  pentora plugin remove ssh-cve-2024-6387
  pentora plugin rm ssh-cve-2024-6387

  # JSON output
  pentora plugin uninstall ssh --output json`,
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

			// Bind flags to options (centralized binding)
			opts, err := bind.BindUninstallOptions(cmd)
			if err != nil {
				return err
			}

			// Determine target plugin name (if specific plugin)
			var target string
			if len(args) > 0 {
				target = args[0]
			}

			// Call service layer
			result, err := svc.Uninstall(ctx, target, opts)
			if err != nil {
				return formatter.PrintError(err)
			}

			// Print results
			return printUninstallResult(formatter, result)
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().Bool("all", false, "Uninstall all plugins")
	cmd.Flags().String("category", "", "Uninstall all plugins from category (ssh, http, tls, database, network)")
	cmd.Flags().String("output", "table", "Output format: json, table")
	cmd.Flags().Bool("quiet", false, "Suppress non-essential output")
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

// printUninstallResult formats and prints the uninstall result using the formatter
func printUninstallResult(f format.Formatter, result *plugin.UninstallResult) error {
	summary := fmt.Sprintf("Uninstall Summary: Removed: %d", result.RemovedCount)
	if result.FailedCount > 0 {
		summary += fmt.Sprintf(", Failed: %d", result.FailedCount)
	}
	if result.RemainingCount > 0 {
		summary += fmt.Sprintf(", Remaining: %d", result.RemainingCount)
	}

	if err := f.PrintSummary(summary); err != nil {
		return err
	}

	if result.RemainingCount == 0 {
		return f.PrintSummary("✓ All plugins uninstalled. Cache is empty.")
	}

	return nil
}
