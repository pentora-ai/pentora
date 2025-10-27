package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/spf13/cobra"
)

func newCleanCommand() *cobra.Command {
	var cacheDir string

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean unused plugin cache entries",
		Long: `Remove old or unused plugin cache entries.

Removes cached plugins older than the specified duration.
Use --dry-run to preview what would be deleted without actually deleting.`,
		Example: `  # Clean cache entries older than 30 days
  pentora plugin clean --older-than 720h

  # Preview what would be deleted
  pentora plugin clean --older-than 720h --dry-run

  # Clean custom cache directory
  pentora plugin clean --older-than 168h --cache-dir /custom/path

  # JSON output
  pentora plugin clean --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			opts, err := bind.BindCleanOptions(cmd)
			if err != nil {
				return err
			}

			// Print header (non-quiet mode)
			if err := formatter.PrintSummary(fmt.Sprintf("Cleaning plugin cache: %s", cacheDir)); err != nil {
				return err
			}
			if err := formatter.PrintSummary(fmt.Sprintf("Removing entries older than: %s", opts.OlderThan)); err != nil {
				return err
			}
			if opts.DryRun {
				if err := formatter.PrintSummary("(Dry run - no files will be deleted)"); err != nil {
					return err
				}
			}

			// Call service layer
			result, err := svc.Clean(cmd.Context(), opts)
			if err != nil {
				return formatter.PrintError(err)
			}

			// Print results
			return printCleanResult(formatter, result, opts.DryRun)
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().Bool("dry-run", false, "Preview what would be deleted without actually deleting")
	cmd.Flags().String("older-than", "720h", "Remove cache entries older than this duration (e.g., 720h for 30 days)")
	cmd.Flags().String("output", "table", "Output format: json, table")
	cmd.Flags().Bool("quiet", false, "Suppress non-essential output")
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

// printCleanResult formats and prints the clean result using the formatter
func printCleanResult(f format.Formatter, result *plugin.CleanResult, dryRun bool) error {
	if result.RemovedCount == 0 {
		return f.PrintSummary("No old plugin cache entries found to remove.")
	}

	var summary string
	if dryRun {
		summary = fmt.Sprintf("[DRY RUN] Would remove %d cache entries", result.RemovedCount)
		if result.Freed > 0 {
			summary += fmt.Sprintf(", would free %s", formatBytes(result.Freed))
		}
		if err := f.PrintSummary(summary); err != nil {
			return err
		}
		return f.PrintSummary("Run without --dry-run to actually delete these files.")
	}

	summary = fmt.Sprintf("Removed %d cache entries", result.RemovedCount)
	if result.Freed > 0 {
		summary += fmt.Sprintf(", freed %s", formatBytes(result.Freed))
	}
	return f.PrintSummary(summary)
}
