package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/rs/zerolog/log"
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

			// Create cache manager
			cacheManager, err := plugin.NewCacheManager(cacheDir)
			if err != nil {
				return fmt.Errorf("create cache manager: %w", err)
			}

			// Calculate size before cleaning
			sizeBefore, err := cacheManager.Size()
			if err != nil {
				log.Debug().Err(err).Msg("Failed to calculate cache size before cleaning")
			}

			// Run prune operation
			fmt.Printf("Cleaning plugin cache: %s\n", cacheDir)
			fmt.Printf("Removing entries older than: %s\n", duration)
			if dryRun {
				fmt.Println("(Dry run - no files will be deleted)")
			}
			fmt.Println()

			removed, err := cacheManager.Prune(duration)
			if err != nil {
				return fmt.Errorf("clean cache: %w", err)
			}

			if removed == 0 {
				fmt.Println("No old plugin cache entries found to remove.")
				return nil
			}

			// Calculate size after cleaning
			sizeAfter, err := cacheManager.Size()
			if err != nil {
				log.Debug().Err(err).Msg("Failed to calculate cache size after cleaning")
			}

			freed := sizeBefore - sizeAfter

			fmt.Println()
			if dryRun {
				fmt.Printf("Would remove %d cache entries.\n", removed)
				if freed > 0 {
					fmt.Printf("Would free approximately %s of disk space.\n", formatBytes(freed))
				}
				fmt.Println("\nRun without --dry-run to actually delete these files.")
			} else {
				fmt.Printf("Removed %d cache entries.\n", removed)
				if freed > 0 {
					fmt.Printf("Freed %s of disk space.\n", formatBytes(freed))
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview what would be deleted without actually deleting")
	cmd.Flags().StringVar(&olderThan, "older-than", "720h", "Remove cache entries older than this duration (e.g., 720h for 30 days)")

	return cmd
}
