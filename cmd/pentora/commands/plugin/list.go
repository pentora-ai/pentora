package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var (
		cacheDir string
		verbose  bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all installed plugins",
		Long: `List all plugins currently installed in the cache.

Displays plugin name, version, and installation status for each plugin
found in the plugin cache directory.`,
		Example: `  # List all installed plugins
  pentora plugin list

  # List plugins with detailed information
  pentora plugin list --verbose

  # List plugins from custom cache directory
  pentora plugin list --cache-dir /custom/path`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use default cache dir if not specified
			if cacheDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("get home directory: %w", err)
				}
				cacheDir = filepath.Join(homeDir, ".pentora", "plugins", "cache")
			}

			// Create manifest manager
			manifestPath := filepath.Join(filepath.Dir(cacheDir), "registry.json")
			manifestMgr, err := plugin.NewManifestManager(manifestPath)
			if err != nil {
				return fmt.Errorf("create manifest manager: %w", err)
			}

			// List all installed plugins
			entries, err := manifestMgr.List()
			if err != nil {
				return fmt.Errorf("list plugins: %w", err)
			}

			if len(entries) == 0 {
				fmt.Println("No plugins installed.")
				fmt.Println()
				fmt.Println("To install plugins, use:")
				fmt.Println("  pentora plugin install <plugin-name>")
				return nil
			}

			// Print results
			fmt.Printf("Found %d plugin(s):\n\n", len(entries))

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			if verbose {
				fmt.Fprintln(w, "NAME\tVERSION\tCHECKSUM\tDOWNLOAD URL")
				fmt.Fprintln(w, "----\t-------\t--------\t------------")
				for _, entry := range entries {
					if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
						entry.Name,
						entry.Version,
						truncateChecksum(entry.Checksum),
						truncateURL(entry.DownloadURL)); err != nil {
						log.Debug().Err(err).Msg("Failed to write plugin entry")
					}
				}
			} else {
				fmt.Fprintln(w, "NAME\tVERSION")
				fmt.Fprintln(w, "----\t-------")
				for _, entry := range entries {
					if _, err := fmt.Fprintf(w, "%s\t%s\n", entry.ID, entry.Version); err != nil {
						log.Debug().Err(err).Msg("Failed to write plugin entry")
					}
				}
			}
			if err := w.Flush(); err != nil {
				log.Warn().Err(err).Msg("Failed to flush output")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed information")

	return cmd
}

// truncateChecksum truncates checksum for display (shows first 12 chars)
func truncateChecksum(checksum string) string {
	if len(checksum) > 12 {
		return checksum[:12] + "..."
	}
	return checksum
}

// truncateURL truncates URL for display (shows last 40 chars)
func truncateURL(url string) string {
	if len(url) > 40 {
		return "..." + url[len(url)-37:]
	}
	return url
}
