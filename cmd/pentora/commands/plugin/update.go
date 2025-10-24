package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	var (
		cacheDir        string
		source          string
		category        string
		dryRun          bool
		forceRedownload bool
	)

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
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("get home directory: %w", err)
				}
				cacheDir = filepath.Join(homeDir, ".pentora", "plugins", "cache")
			}

			// Create cache manager
			cacheManager, err := plugin.NewCacheManager(cacheDir)
			if err != nil {
				return fmt.Errorf("create cache manager: %w", err)
			}

			// Create downloader with default sources
			sources := []plugin.PluginSource{
				{
					Name:     "official",
					URL:      "https://plugins.pentora.ai/manifest.yaml",
					Enabled:  true,
					Priority: 1,
					Mirrors: []string{
						"https://raw.githubusercontent.com/pentora-ai/pentora-plugins/main/manifest.yaml",
					},
				},
			}

			// Filter by source if specified
			if source != "" {
				var filteredSources []plugin.PluginSource
				for _, s := range sources {
					if s.Name == source {
						filteredSources = append(filteredSources, s)
					}
				}
				if len(filteredSources) == 0 {
					return fmt.Errorf("source '%s' not found", source)
				}
				sources = filteredSources
			}

			downloader := plugin.NewDownloader(cacheManager, plugin.WithSources(sources))

			// Fetch manifest from each source
			fmt.Println("Fetching plugin manifests...")
			var allPlugins []plugin.PluginManifestEntry
			for _, src := range sources {
				if !src.Enabled {
					continue
				}

				fmt.Printf("  Fetching from %s...\n", src.Name)
				manifest, err := downloader.FetchManifest(ctx, src)
				if err != nil {
					log.Warn().
						Str("source", src.Name).
						Err(err).
						Msg("Failed to fetch manifest from source")
					continue
				}

				fmt.Printf("  ✓ Found %d plugins from %s\n", len(manifest.Plugins), src.Name)
				allPlugins = append(allPlugins, manifest.Plugins...)
			}

			if len(allPlugins) == 0 {
				fmt.Println("\nNo plugins found in any source.")
				return nil
			}

			// Filter by category if specified
			if category != "" {
				var filteredPlugins []plugin.PluginManifestEntry
				targetCategory := plugin.Category(category)
				for _, p := range allPlugins {
					for _, cat := range p.Categories {
						if cat == targetCategory {
							filteredPlugins = append(filteredPlugins, p)
							break
						}
					}
				}
				allPlugins = filteredPlugins
				if len(allPlugins) == 0 {
					fmt.Printf("\nNo plugins found in category '%s'\n", category)
					return nil
				}
			}

			// Dry run: just show what would be downloaded
			if dryRun {
				fmt.Printf("\n[DRY RUN] Would download %d plugin(s):\n\n", len(allPlugins))
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "NAME\tVERSION\tCATEGORY\tSIZE")
				fmt.Fprintln(w, "----\t-------\t--------\t----")
				for _, p := range allPlugins {
					categoryStr := ""
					if len(p.Categories) > 0 {
						categoryStr = string(p.Categories[0])
					}
					sizeKB := float64(p.Size) / 1024.0
					if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%.1f KB\n",
						p.Name, p.Version, categoryStr, sizeKB); err != nil {
						log.Debug().Err(err).Msg("Failed to write plugin entry")
					}
				}
				if err := w.Flush(); err != nil {
					log.Warn().Err(err).Msg("Failed to flush output")
				}
				return nil
			}

			// Download plugins
			fmt.Printf("\nDownloading %d plugin(s)...\n\n", len(allPlugins))
			downloadedCount := 0
			skippedCount := 0
			failedCount := 0

			for _, p := range allPlugins {
				// Check if already cached (unless force re-download)
				if !forceRedownload {
					if _, err := cacheManager.GetEntry(p.Name, p.Version); err == nil {
						skippedCount++
						log.Debug().
							Str("plugin", p.Name).
							Str("version", p.Version).
							Msg("Plugin already cached, skipping")
						continue
					}
				}

				fmt.Printf("  Downloading %s v%s...", p.Name, p.Version)

				_, err := downloader.Download(ctx, p.Name, p.Version)
				if err != nil {
					fmt.Printf(" ✗\n")
					log.Warn().
						Str("plugin", p.Name).
						Err(err).
						Msg("Failed to download plugin")
					failedCount++
					continue
				}

				fmt.Printf(" ✓\n")
				downloadedCount++
			}

			// Summary
			fmt.Printf("\nUpdate Summary:\n")
			fmt.Printf("  Downloaded: %d\n", downloadedCount)
			fmt.Printf("  Skipped (already cached): %d\n", skippedCount)
			if failedCount > 0 {
				fmt.Printf("  Failed: %d\n", failedCount)
			}
			fmt.Printf("  Total plugins in cache: %d\n", downloadedCount+skippedCount)

			if downloadedCount > 0 {
				fmt.Printf("\nPlugins stored in: %s\n", cacheDir)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")
	cmd.Flags().StringVar(&source, "source", "", "Download from specific source (e.g., 'official')")
	cmd.Flags().StringVar(&category, "category", "", "Download only plugins from category (ssh, http, tls, database, network)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be downloaded without downloading")
	cmd.Flags().BoolVar(&forceRedownload, "force", false, "Force re-download even if already cached")

	return cmd
}
