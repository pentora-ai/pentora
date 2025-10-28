package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
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
			return executeUpdateCommand(cmd, cacheDir)
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

// executeUpdateCommand orchestrates the update command execution
func executeUpdateCommand(cmd *cobra.Command, cacheDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Setup dependencies
	formatter := getFormatter(cmd)
	svc, err := getPluginService(cacheDir)
	if err != nil {
		return err
	}

	// Bind flags to options
	opts, err := bind.BindUpdateOptions(cmd)
	if err != nil {
		return err
	}

	// Call service layer
	result, err := svc.Update(ctx, opts)

	// Handle partial failure (exit code 8)
	if handleErr := handlePartialFailure(err, formatter, func() error {
		return printUpdateResult(formatter, result, opts.DryRun)
	}); handleErr != nil {
		return handleErr
	}

	// Handle total failure
	if err != nil {
		return formatter.PrintError(err)
	}

	// Print results
	return printUpdateResult(formatter, result, opts.DryRun)
}

// printUpdateResult formats and prints the update result
func printUpdateResult(f format.Formatter, result *plugin.UpdateResult, dryRun bool) error {
	if f.IsJSON() {
		return printUpdateJSON(f, result, dryRun)
	}

	// Dry run mode
	if dryRun {
		return printUpdateDryRun(f, result)
	}

	// Print table
	rows := buildPluginTable(result.Plugins)
	if len(rows) > 0 {
		if err := f.PrintTable([]string{"Name", "Version", "Category"}, rows); err != nil {
			return err
		}
	}

	// Print summary
	if err := f.PrintSummary(buildUpdateSummary(result)); err != nil {
		return err
	}

	// Print errors if any
	if err := printErrorList(f, result.Errors); err != nil {
		return err
	}

	// Success message
	if result.UpdatedCount > 0 && result.FailedCount == 0 {
		return f.PrintSummary("\n✓ Plugins updated successfully")
	}

	return nil
}

// printUpdateJSON outputs update result as JSON
func printUpdateJSON(f format.Formatter, result *plugin.UpdateResult, dryRun bool) error {
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

// printUpdateDryRun prints dry run output
func printUpdateDryRun(f format.Formatter, result *plugin.UpdateResult) error {
	rows := buildPluginTable(result.Plugins)
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

// buildUpdateSummary builds the summary message for update results
func buildUpdateSummary(result *plugin.UpdateResult) string {
	summary := fmt.Sprintf("Update Summary: Downloaded: %d, Skipped: %d", result.UpdatedCount, result.SkippedCount)
	if result.FailedCount > 0 {
		summary += fmt.Sprintf(", Failed: %d", result.FailedCount)
	}
	summary += fmt.Sprintf(", Total in cache: %d", result.UpdatedCount+result.SkippedCount)
	return summary
}
