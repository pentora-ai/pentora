package plugin

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/spf13/cobra"
)

func newUninstallCommand() *cobra.Command {
	var (
		cacheDir string
		all      bool
		category string
	)

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
  pentora plugin rm ssh-cve-2024-6387`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

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

			// Build uninstall options
			opts := plugin.UninstallOptions{
				All: all,
			}

			if category != "" {
				opts.Category = plugin.Category(category)
			}

			// Determine target plugin name (if specific plugin)
			var target string
			if len(args) > 0 {
				target = args[0]
			}

			// Call service layer
			result, err := svc.Uninstall(ctx, target, opts)
			if err != nil {
				return err
			}

			// Print results
			printUninstallResult(result)

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().BoolVar(&all, "all", false, "Uninstall all plugins")
	cmd.Flags().StringVar(&category, "category", "", "Uninstall all plugins from category (ssh, http, tls, database, network)")

	return cmd
}

// printUninstallResult formats and prints the uninstall result
func printUninstallResult(result *plugin.UninstallResult) {
	fmt.Printf("\nUninstall Summary:\n")
	fmt.Printf("  Removed: %d\n", result.RemovedCount)
	if result.FailedCount > 0 {
		fmt.Printf("  Failed: %d\n", result.FailedCount)
	}

	if result.RemainingCount > 0 {
		fmt.Printf("  Remaining in cache: %d\n", result.RemainingCount)
	} else {
		fmt.Println("\n✓ All plugins uninstalled. Cache is empty.")
	}
}
