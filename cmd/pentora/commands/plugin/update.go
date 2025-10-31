package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
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

	// Setup structured logger
	logger := log.With().
		Str("component", "plugin.cli").
		Str("op", "update").
		Logger()

	start := time.Now()
	defer func() {
		logger.Info().
			Dur("duration_ms", time.Since(start)).
			Msg("update completed")
	}()

	// Log operation start with request snapshot
	logger.Info().
		Str("source", opts.Source).
		Str("category", string(opts.Category)).
		Bool("dry_run", opts.DryRun).
		Bool("force", opts.Force).
		Msg("update started")

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
		return formatter.PrintTotalFailureSummary("update", err, plugin.ErrorCode(err))
	}

	// Log success with metrics
	logger.Info().
		Int("updated_count", result.UpdatedCount).
		Int("skipped_count", result.SkippedCount).
		Int("failed_count", result.FailedCount).
		Bool("dry_run", opts.DryRun).
		Msg("update succeeded")

	// Print results
	return printUpdateResult(formatter, result, opts.DryRun)
}

// printUpdateResult formats and prints the update result
func printUpdateResult(f format.Formatter, result *plugin.UpdateResult, dryRun bool) error {
	if f.IsJSON() {
		return printUpdateJSON(f, result, dryRun)
	}

	if dryRun {
		return printUpdateDryRun(f, result)
	}

	// Success case
	if result.FailedCount == 0 && result.UpdatedCount > 0 {
		return printUpdateSuccess(f, result)
	}

	// Partial failure
	if result.UpdatedCount > 0 && result.FailedCount > 0 {
		return printUpdatePartialFailure(f, result)
	}

	// All skipped
	if result.SkippedCount > 0 && result.UpdatedCount == 0 && result.FailedCount == 0 {
		return f.PrintSummary("All plugins are already up-to-date")
	}

	// Total failure
	if result.FailedCount > 0 && result.UpdatedCount == 0 {
		return printUpdateTotalFailure(f, result)
	}

	return nil
}

// printUpdateSuccess prints success result for update operation
func printUpdateSuccess(f format.Formatter, result *plugin.UpdateResult) error {
	if len(result.Plugins) == 1 {
		p := result.Plugins[0]
		return f.PrintSuccessSummary("updated", p.ID, p.Version)
	}
	rows := buildPluginTable(result.Plugins)
	if err := f.PrintTable([]string{"Name", "Version", "Category"}, rows); err != nil {
		return err
	}
	return f.PrintSuccessSummary("updated", fmt.Sprintf("%d plugins", result.UpdatedCount), "")
}

// printUpdatePartialFailure prints partial failure result for update operation
func printUpdatePartialFailure(f format.Formatter, result *plugin.UpdateResult) error {
	rows := buildPluginTable(result.Plugins)
	if len(rows) > 0 {
		if err := f.PrintTable([]string{"Name", "Version", "Category"}, rows); err != nil {
			return err
		}
	}

	errorDetails := convertPluginErrors(result.Errors)
	summary := format.Summary{
		Operation:   "update",
		Success:     result.UpdatedCount,
		Skipped:     result.SkippedCount,
		Failed:      result.FailedCount,
		Errors:      errorDetails,
		TotalErrors: len(result.Errors),
	}
	return f.PrintPartialFailureSummary(summary)
}

// printUpdateTotalFailure prints total failure result for update operation
func printUpdateTotalFailure(f format.Formatter, result *plugin.UpdateResult) error {
	if len(result.Errors) == 1 {
		e := result.Errors[0]
		return f.PrintTotalFailureSummary("update", fmt.Errorf("%s", e.Error), e.Code)
	}
	errorDetails := convertPluginErrors(result.Errors)
	summary := format.Summary{
		Operation:   "update",
		Success:     0,
		Skipped:     result.SkippedCount,
		Failed:      result.FailedCount,
		Errors:      errorDetails,
		TotalErrors: len(result.Errors),
	}
	return f.PrintPartialFailureSummary(summary)
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
