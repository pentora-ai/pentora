package plugin

import (
	"context"
	"errors"
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

			// Handle partial failure (exit code 8)
			if err != nil && errors.Is(err, plugin.ErrPartialFailure) {
				// Print result even on partial failure
				if printErr := printUninstallResult(formatter, result); printErr != nil {
					return printErr
				}
				// Exit with code 8 for partial failure
				os.Exit(plugin.ExitCode(err))
			}

			// Handle total failure (exit code 1, 2, 4, 7, etc.)
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
	// JSON mode: output complete result as JSON
	if f.IsJSON() {
		jsonResult := map[string]any{
			"removed_count":   result.RemovedCount,
			"failed_count":    result.FailedCount,
			"remaining_count": result.RemainingCount,
			"success":         result.FailedCount == 0,
			"partial_failure": result.FailedCount > 0 && result.RemovedCount > 0,
			"errors":          result.Errors,
		}
		return f.PrintJSON(jsonResult)
	}

	// Table mode: use existing summary pattern
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

	// Print errors if any (show first 5, truncate rest)
	// nolint:dupl // Intentional code reuse across install/update/uninstall commands
	if len(result.Errors) > 0 {
		if err := f.PrintSummary("\nFailed plugins:"); err != nil {
			return err
		}

		maxErrors := 5
		for i, e := range result.Errors {
			if i >= maxErrors {
				remaining := len(result.Errors) - maxErrors
				if err := f.PrintSummary(fmt.Sprintf("  ... and %d more (use --output json for full list)", remaining)); err != nil {
					return err
				}
				break
			}
			if err := f.PrintSummary(fmt.Sprintf("  - %s: %s", e.PluginID, e.Error)); err != nil {
				return err
			}
		}

		// Print suggestions
		if err := f.PrintSummary("\n💡 Suggestions:"); err != nil {
			return err
		}

		// Collect unique suggestions
		suggestions := make(map[string]bool)
		for _, e := range result.Errors {
			if e.Suggestion != "" {
				suggestions[e.Suggestion] = true
			}
		}

		for suggestion := range suggestions {
			if err := f.PrintSummary(fmt.Sprintf("  → %s", suggestion)); err != nil {
				return err
			}
		}
	}

	// Success message
	if result.RemovedCount > 0 && result.FailedCount == 0 {
		if result.RemainingCount == 0 {
			return f.PrintSummary("\n✓ All plugins uninstalled. Cache is empty.")
		}
		return f.PrintSummary("\n✓ Plugins uninstalled successfully")
	}

	return nil
}
