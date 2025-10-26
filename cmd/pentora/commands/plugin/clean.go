package plugin

import (
	"fmt"
	"path/filepath"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
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
  pentora plugin clean --older-than 168h --cache-dir /custom/path`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			// Call service layer
			fmt.Printf("Cleaning plugin cache: %s\n", cacheDir)
			fmt.Printf("Removing entries older than: %s\n", opts.OlderThan)
			if opts.DryRun {
				fmt.Println("(Dry run - no files will be deleted)")
			}
			fmt.Println()

			result, err := svc.Clean(cmd.Context(), opts)
			if err != nil {
				return err
			}

			// Print results
			printCleanResult(result, opts.DryRun)

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().Bool("dry-run", false, "Preview what would be deleted without actually deleting")
	cmd.Flags().String("older-than", "720h", "Remove cache entries older than this duration (e.g., 720h for 30 days)")

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
