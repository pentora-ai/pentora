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

func newInstallCommand() *cobra.Command {
	var cacheDir string

	cmd := &cobra.Command{
		Use:   "install <category|plugin-name>",
		Short: "Install plugins by category or name",
		Long: `Install plugins from remote repositories by category or specific plugin name.

This command downloads plugins from configured sources and stores them in the local cache.
You can install entire categories (ssh, http, tls, database, network) or specific plugins by name.`,
		Example: `  # Install all SSH plugins
  pentora plugin install ssh

  # Install all HTTP plugins
  pentora plugin install http

  # Install specific plugin by name
  pentora plugin install ssh-cve-2024-6387

  # Install from specific source
  pentora plugin install ssh --source official

  # Force re-install even if already cached
  pentora plugin install ssh --force

  # JSON output
  pentora plugin install ssh --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeInstallCommand(cmd, args[0], cacheDir)
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().String("source", "", "Install from specific source (e.g., 'official')")
	cmd.Flags().Bool("force", false, "Force re-install even if already cached")
	cmd.Flags().String("output", "table", "Output format: json, table")
	cmd.Flags().Bool("quiet", false, "Suppress non-essential output")
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

// executeInstallCommand orchestrates the install command execution
func executeInstallCommand(cmd *cobra.Command, target, cacheDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Bind flags to options early for logging
	opts, err := bind.BindInstallOptions(cmd)
	if err != nil {
		return err
	}

	// Setup structured logger with operation context
	logger := log.With().
		Str("component", "plugin.cli").
		Str("op", "install").
		Logger()

	start := time.Now()
	defer func() {
		logger.Info().
			Dur("duration_ms", time.Since(start)).
			Msg("install completed")
	}()

	// Log operation start with request snapshot
	logger.Info().
		Str("target", target).
		Str("source", opts.Source).
		Bool("force", opts.Force).
		Msg("install started")

	// Setup dependencies
	formatter := getFormatter(cmd)
	svc, err := getPluginService(cacheDir)
	if err != nil {
		return err
	}

	// Call service layer
	result, err := svc.Install(ctx, target, opts)
	// Handle errors with structured logging
	if err != nil {

		// Handle partial failure (exit code 8)
		if handleErr := handlePartialFailure(err, formatter, func() error {
			return printInstallResult(formatter, result)
		}); handleErr != nil {
			return handleErr
		}

		// Handle total failure
		return formatter.PrintTotalFailureSummary("install", err, plugin.ErrorCode(err))
	}

	// Log success with result metrics
	logger.Info().
		Int("installed_count", result.InstalledCount).
		Int("skipped_count", result.SkippedCount).
		Int("failed_count", result.FailedCount).
		Msg("install succeeded")

	// Print results
	return printInstallResult(formatter, result)
}

// printInstallResult formats and prints the install result
func printInstallResult(f format.Formatter, result *plugin.InstallResult) error {
	if f.IsJSON() {
		return printInstallJSON(f, result)
	}

	// Success case: no failures
	if result.FailedCount == 0 && result.InstalledCount > 0 {
		// Single plugin installed
		if len(result.Plugins) == 1 {
			p := result.Plugins[0]
			return f.PrintSuccessSummary("installed", p.ID, p.Version)
		}
		// Multiple plugins installed - show table and summary
		rows := buildPluginTable(result.Plugins)
		if err := f.PrintTable([]string{"Name", "Version", "Category"}, rows); err != nil {
			return err
		}
		return f.PrintSuccessSummary("installed", fmt.Sprintf("%d plugins", result.InstalledCount), "")
	}

	// Partial failure case: some succeeded, some failed
	if result.InstalledCount > 0 && result.FailedCount > 0 {
		// Show table of successful installs
		rows := buildPluginTable(result.Plugins)
		if len(rows) > 0 {
			if err := f.PrintTable([]string{"Name", "Version", "Category"}, rows); err != nil {
				return err
			}
		}

		// Print partial failure summary
		errorDetails := convertPluginErrors(result.Errors)
		summary := format.Summary{
			Operation:   "install",
			Success:     result.InstalledCount,
			Skipped:     result.SkippedCount,
			Failed:      result.FailedCount,
			Errors:      errorDetails,
			TotalErrors: len(result.Errors),
		}
		return f.PrintPartialFailureSummary(summary)
	}

	// All skipped (already installed)
	if result.SkippedCount > 0 && result.InstalledCount == 0 && result.FailedCount == 0 {
		return f.PrintSummary("All plugins already installed (use --force to reinstall)")
	}

	// Total failure case: all failed
	if result.FailedCount > 0 && result.InstalledCount == 0 {
		// Single error - show detailed failure
		if len(result.Errors) == 1 {
			e := result.Errors[0]
			return f.PrintTotalFailureSummary("install", fmt.Errorf("%s", e.Error), e.Code)
		}

		// Multiple errors - show as partial failure summary
		errorDetails := convertPluginErrors(result.Errors)
		summary := format.Summary{
			Operation:   "install",
			Success:     0,
			Skipped:     result.SkippedCount,
			Failed:      result.FailedCount,
			Errors:      errorDetails,
			TotalErrors: len(result.Errors),
		}
		return f.PrintPartialFailureSummary(summary)
	}

	return nil
}

// printInstallJSON outputs install result as JSON
func printInstallJSON(f format.Formatter, result *plugin.InstallResult) error {
	jsonResult := map[string]any{
		"plugins":         result.Plugins,
		"installed_count": result.InstalledCount,
		"skipped_count":   result.SkippedCount,
		"failed_count":    result.FailedCount,
		"success":         result.FailedCount == 0,
		"partial_failure": result.FailedCount > 0 && result.InstalledCount > 0,
		"errors":          result.Errors,
	}
	return f.PrintJSON(jsonResult)
}
