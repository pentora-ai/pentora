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

	// Setup dependencies
	formatter := getFormatter(cmd)
	svc, err := getPluginService(cacheDir)
	if err != nil {
		return err
	}

	// Bind flags to options
	opts, err := bind.BindInstallOptions(cmd)
	if err != nil {
		return err
	}

	// Call service layer
	result, err := svc.Install(ctx, target, opts)

	// Handle partial failure (exit code 8)
	if handleErr := handlePartialFailure(err, formatter, func() error {
		return printInstallResult(formatter, result)
	}); handleErr != nil {
		return handleErr
	}

	// Handle total failure
	if err != nil {
		return formatter.PrintError(err)
	}

	// Print results
	return printInstallResult(formatter, result)
}

// printInstallResult formats and prints the install result
func printInstallResult(f format.Formatter, result *plugin.InstallResult) error {
	if f.IsJSON() {
		return printInstallJSON(f, result)
	}

	// Print table
	rows := buildPluginTable(result.Plugins)
	if len(rows) > 0 {
		if err := f.PrintTable([]string{"Name", "Version", "Category"}, rows); err != nil {
			return err
		}
	}

	// Print summary
	if err := f.PrintSummary(buildInstallSummary(result)); err != nil {
		return err
	}

	// Print errors if any
	if err := printErrorList(f, result.Errors); err != nil {
		return err
	}

	// Success message
	if result.InstalledCount > 0 && result.FailedCount == 0 {
		return f.PrintSummary("\n✓ Plugins installed successfully")
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

// buildInstallSummary builds the summary message for install results
func buildInstallSummary(result *plugin.InstallResult) string {
	summary := fmt.Sprintf("Installation Summary: Installed: %d", result.InstalledCount)
	if result.SkippedCount > 0 {
		summary += fmt.Sprintf(", Already installed: %d", result.SkippedCount)
	}
	if result.FailedCount > 0 {
		summary += fmt.Sprintf(", Failed: %d", result.FailedCount)
	}
	return summary
}
