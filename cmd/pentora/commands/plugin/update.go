package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/spf13/cobra"
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

	if result.UpdatedCount > 0 {
		return f.PrintSummary("✓ Plugins updated successfully")
	}

	return nil
}
