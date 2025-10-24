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

func newInstallCommand() *cobra.Command {
	var (
		cacheDir string
		source   string
		force    bool
	)

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
  pentora plugin install ssh --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
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
			fmt.Printf("Fetching plugin manifests from %s...\n", sources[0].Name)
			var allPlugins []plugin.PluginManifestEntry
			for _, src := range sources {
				if !src.Enabled {
					continue
				}

				manifest, err := downloader.FetchManifest(ctx, src)
				if err != nil {
					log.Warn().
						Str("source", src.Name).
						Err(err).
						Msg("Failed to fetch manifest from source")
					continue
				}

				allPlugins = append(allPlugins, manifest.Plugins...)
			}

			if len(allPlugins) == 0 {
				return fmt.Errorf("no plugins found in any source")
			}

			// Check if target is a category or plugin name
			isCategory := plugin.Category(target).IsValid()
			var toInstall []plugin.PluginManifestEntry

			if isCategory {
				// Install entire category
				targetCategory := plugin.Category(target)
				for _, p := range allPlugins {
					for _, cat := range p.Categories {
						if cat == targetCategory {
							toInstall = append(toInstall, p)
							break
						}
					}
				}
				if len(toInstall) == 0 {
					return fmt.Errorf("no plugins found in category '%s'", target)
				}
				fmt.Printf("Found %d plugin(s) in category '%s'\n", len(toInstall), target)
			} else {
				// Install specific plugin by name
				found := false
				for _, p := range allPlugins {
					if p.Name == target {
						toInstall = append(toInstall, p)
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("plugin '%s' not found in any source", target)
				}
				fmt.Printf("Found plugin '%s'\n", target)
			}

			// Show plugins to install
			fmt.Println("\nPlugins to install:")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tVERSION\tCATEGORY\tSIZE")
			fmt.Fprintln(w, "----\t-------\t--------\t----")
			for _, p := range toInstall {
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

			// Download plugins
			fmt.Printf("\nInstalling %d plugin(s)...\n\n", len(toInstall))
			downloadedCount := 0
			skippedCount := 0
			failedCount := 0

			for _, p := range toInstall {
				// Check if already cached (unless force re-install)
				if !force {
					if _, err := cacheManager.GetEntry(p.Name, p.Version); err == nil {
						skippedCount++
						fmt.Printf("  %s v%s already installed (use --force to reinstall)\n", p.Name, p.Version)
						continue
					}
				}

				fmt.Printf("  Installing %s v%s...", p.Name, p.Version)

				_, err := downloader.Download(ctx, p.Name, p.Version)
				if err != nil {
					fmt.Printf(" ✗\n")
					log.Warn().
						Str("plugin", p.Name).
						Err(err).
						Msg("Failed to install plugin")
					failedCount++
					continue
				}

				fmt.Printf(" ✓\n")
				downloadedCount++
			}

			// Summary
			fmt.Printf("\nInstallation Summary:\n")
			fmt.Printf("  Installed: %d\n", downloadedCount)
			if skippedCount > 0 {
				fmt.Printf("  Already installed: %d\n", skippedCount)
			}
			if failedCount > 0 {
				fmt.Printf("  Failed: %d\n", failedCount)
			}

			if downloadedCount > 0 {
				fmt.Printf("\n✓ Plugins installed successfully in: %s\n", cacheDir)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")
	cmd.Flags().StringVar(&source, "source", "", "Install from specific source (e.g., 'official')")
	cmd.Flags().BoolVar(&force, "force", false, "Force re-install even if already cached")

	return cmd
}
