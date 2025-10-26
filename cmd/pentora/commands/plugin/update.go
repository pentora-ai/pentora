package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/rs/zerolog/log"
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
  pentora plugin update --source official`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

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
				return err
			}

			// Print results
			printUpdateResult(result, opts.DryRun)

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().String("source", "", "Download from specific source (e.g., 'official')")
	cmd.Flags().String("category", "", "Download only plugins from category (ssh, http, tls, database, network)")
	cmd.Flags().Bool("dry-run", false, "Show what would be downloaded without downloading")
	cmd.Flags().Bool("force", false, "Force re-download even if already cached")

	return cmd
}

// printUpdateResult formats and prints the update result
func printUpdateResult(result *plugin.UpdateResult, dryRun bool) {
	// Dry run: show what would be downloaded
	if dryRun {
		fmt.Printf("\n[DRY RUN] Would download %d plugin(s):\n\n", len(result.Plugins))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tCATEGORY")
		fmt.Fprintln(w, "----\t-------\t--------")
		for _, p := range result.Plugins {
			categoryStr := ""
			if len(p.Tags) > 0 {
				categoryStr = p.Tags[0]
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n",
				p.Name, p.Version, categoryStr); err != nil {
				log.Debug().Err(err).Msg("Failed to write plugin entry")
			}
		}
		if err := w.Flush(); err != nil {
			log.Warn().Err(err).Msg("Failed to flush output")
		}
		return
	}

	// Summary
	fmt.Printf("\nUpdate Summary:\n")
	fmt.Printf("  Downloaded: %d\n", result.UpdatedCount)
	fmt.Printf("  Skipped (already cached): %d\n", result.SkippedCount)
	if result.FailedCount > 0 {
		fmt.Printf("  Failed: %d\n", result.FailedCount)
	}
	fmt.Printf("  Total plugins in cache: %d\n", result.UpdatedCount+result.SkippedCount)

	if result.UpdatedCount > 0 {
		fmt.Printf("\n✓ Plugins updated successfully\n")
	}
}
