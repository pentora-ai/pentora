package plugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
)

func newUpdateCommand() *cobra.Command {
	var cacheDir string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update plugins from remote repositories",
		Long: `Download and update plugins from remote plugin repositories.

This command fetches the latest plugin manifest from configured sources and downloads
new or updated plugins to the local cache. By default, it downloads all core plugins.`,
		Example: `  # Update all plugins from default source
  pentora plugin update

  # Update only SSH plugins
  pentora plugin update --category ssh

  # Dry run to see what would be downloaded
  pentora plugin update --dry-run

  # Force re-download even if cached
  pentora plugin update --force

  # Update from specific source
  pentora plugin update --source official

  # JSON output
  pentora plugin update --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

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
			opts, err := bind.BindUpdateOptions(cmd)
			if err != nil {
				return err
			}

			// Call service layer
			result, err := svc.Update(ctx, opts)

			// Handle partial failure (exit code 8)
			if err != nil && errors.Is(err, plugin.ErrPartialFailure) {
				// Print result even on partial failure
				if printErr := printUpdateResult(formatter, result, opts.DryRun); printErr != nil {
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
			return printUpdateResult(formatter, result, opts.DryRun)
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().String("source", "", "Download from specific source (e.g., 'official')")
	cmd.Flags().String("category", "", "Download only plugins from category (ssh, http, tls, database, network)")
	cmd.Flags().Bool("dry-run", false, "Show what would be downloaded without downloading")
	cmd.Flags().Bool("force", false, "Force re-download even if already cached")
	cmd.Flags().String("output", "table", "Output format: json, table")
	cmd.Flags().Bool("quiet", false, "Suppress non-essential output")
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

// printUpdateResult formats and prints the update result using the formatter
func printUpdateResult(f format.Formatter, result *plugin.UpdateResult, dryRun bool) error {
	// JSON mode: output complete result as JSON
	if f.IsJSON() {
		jsonResult := map[string]any{
			"plugins":         result.Plugins,
			"updated_count":   result.UpdatedCount,
			"skipped_count":   result.SkippedCount,
			"failed_count":    result.FailedCount,
			"dry_run":         dryRun,
			"success":         result.FailedCount == 0,
			"partial_failure": result.FailedCount > 0 && result.UpdatedCount > 0,
			"errors":          result.Errors,
		}
		return f.PrintJSON(jsonResult)
	}

	// Table mode: use existing table + summary pattern
	// Build table rows
	var rows [][]string
	for _, p := range result.Plugins {
		categoryStr := ""
		if len(p.Tags) > 0 {
			categoryStr = p.Tags[0]
		}
		rows = append(rows, []string{p.Name, p.Version, categoryStr})
	}

	// Dry run mode
	if dryRun {
		if len(rows) > 0 {
			if err := f.PrintSummary(fmt.Sprintf("[DRY RUN] Would download %d plugin(s):", len(result.Plugins))); err != nil {
				return err
			}
			if err := f.PrintTable([]string{"Name", "Version", "Category"}, rows); err != nil {
				return err
			}
		}
		return f.PrintSummary("Dry run completed (no changes made)")
	}

	// Print table if plugins were processed
	if len(rows) > 0 {
		if err := f.PrintTable([]string{"Name", "Version", "Category"}, rows); err != nil {
			return err
		}
	}

	// Summary
	summary := fmt.Sprintf("Update Summary: Downloaded: %d, Skipped: %d", result.UpdatedCount, result.SkippedCount)
	if result.FailedCount > 0 {
		summary += fmt.Sprintf(", Failed: %d", result.FailedCount)
	}
	summary += fmt.Sprintf(", Total in cache: %d", result.UpdatedCount+result.SkippedCount)

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
	if result.UpdatedCount > 0 && result.FailedCount == 0 {
		return f.PrintSummary("\n✓ Plugins updated successfully")
	}

	return nil
}
