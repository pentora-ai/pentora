package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/spf13/cobra"
)

func newCleanCommand() *cobra.Command {
	var (
		cacheDir  string
		dryRun    bool
		olderThan string
	)

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
  pentora plugin clean --older-than 168h --cache-dir /custom/path`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use default cache dir if not specified
			if cacheDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("get home directory: %w", err)
				}
				cacheDir = filepath.Join(homeDir, ".pentora", "plugins", "cache")
			}

			// Parse duration
			duration, err := time.ParseDuration(olderThan)
			if err != nil {
				return fmt.Errorf("invalid duration '%s': %w (use format like '720h' for 30 days)", olderThan, err)
			}

			// Create service
			svc, err := plugin.NewService(cacheDir)
			if err != nil {
				return fmt.Errorf("create plugin service: %w", err)
			}

			// Build clean options
			opts := plugin.CleanOptions{
				OlderThan: duration,
				DryRun:    dryRun,
			}

			// Call service layer
			fmt.Printf("Cleaning plugin cache: %s\n", cacheDir)
			fmt.Printf("Removing entries older than: %s\n", duration)
			if dryRun {
				fmt.Println("(Dry run - no files will be deleted)")
			}
			fmt.Println()

			result, err := svc.Clean(cmd.Context(), opts)
			if err != nil {
				return err
			}

			// Print results
			printCleanResult(result, dryRun)

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview what would be deleted without actually deleting")
	cmd.Flags().StringVar(&olderThan, "older-than", "720h", "Remove cache entries older than this duration (e.g., 720h for 30 days)")

	return cmd
}

// printCleanResult formats and prints the clean result
func printCleanResult(result *plugin.CleanResult, dryRun bool) {
	if result.RemovedCount == 0 {
		fmt.Println("No old plugin cache entries found to remove.")
		return
	}

	fmt.Println()
	if dryRun {
		fmt.Printf("Would remove %d cache entries.\n", result.RemovedCount)
		if result.Freed > 0 {
			fmt.Printf("Would free approximately %s of disk space.\n", formatBytes(result.Freed))
		}
		fmt.Println("\nRun without --dry-run to actually delete these files.")
	} else {
		fmt.Printf("Removed %d cache entries.\n", result.RemovedCount)
		if result.Freed > 0 {
			fmt.Printf("Freed %s of disk space.\n", formatBytes(result.Freed))
		}
	}
}
