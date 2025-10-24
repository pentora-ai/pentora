package plugin

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newEmbeddedCommand() *cobra.Command {
	var (
		verbose  bool
		category string
	)

	cmd := &cobra.Command{
		Use:   "embedded",
		Short: "List embedded security check plugins",
		Long: `List all security check plugins embedded in the Pentora binary.

These plugins are compiled into the binary and provide vulnerability detection
for common security issues across SSH, HTTP, TLS, Database, and Network protocols.`,
		Example: `  # List all embedded plugins
  pentora plugin embedded

  # List embedded plugins with detailed information
  pentora plugin embedded --verbose

  # List only SSH plugins
  pentora plugin embedded --category ssh

  # List TLS plugins with details
  pentora plugin embedded --category tls --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load embedded plugins
			plugins, err := plugin.LoadEmbeddedPlugins()
			if err != nil {
				return fmt.Errorf("load embedded plugins: %w", err)
			}

			// Filter by category if specified
			var pluginsToDisplay []*plugin.YAMLPlugin
			categoryFilter := plugin.Category("")
			if category != "" {
				switch category {
				case "ssh":
					categoryFilter = plugin.CategorySSH
				case "http":
					categoryFilter = plugin.CategoryHTTP
				case "tls":
					categoryFilter = plugin.CategoryTLS
				case "database":
					categoryFilter = plugin.CategoryDatabase
				case "network":
					categoryFilter = plugin.CategoryNetwork
				case "misc":
					categoryFilter = plugin.CategoryMisc
				default:
					return fmt.Errorf("unknown category: %s (valid: ssh, http, tls, database, network, misc)", category)
				}

				// Get plugins for the specified category
				if catPlugins, ok := plugins[categoryFilter]; ok {
					pluginsToDisplay = catPlugins
				} else {
					fmt.Printf("No embedded plugins found for category: %s\n", category)
					return nil
				}
			} else {
				// Flatten all plugins
				for _, catPlugins := range plugins {
					pluginsToDisplay = append(pluginsToDisplay, catPlugins...)
				}
			}

			if len(pluginsToDisplay) == 0 {
				fmt.Println("No embedded plugins found.")
				return nil
			}

			// Sort plugins by name for consistent output
			sort.Slice(pluginsToDisplay, func(i, j int) bool {
				return pluginsToDisplay[i].Name < pluginsToDisplay[j].Name
			})

			// Print header
			if category != "" {
				fmt.Printf("Found %d embedded plugin(s) in category '%s':\n\n", len(pluginsToDisplay), category)
			} else {
				fmt.Printf("Found %d embedded plugin(s) total:\n\n", len(pluginsToDisplay))
			}

			// Print table
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			if verbose {
				fmt.Fprintln(w, "NAME\tVERSION\tSEVERITY\tAUTHOR")
				fmt.Fprintln(w, "----\t-------\t--------\t------")
				for _, p := range pluginsToDisplay {
					severity := string(p.Metadata.Severity)
					if severity == "" {
						severity = "info"
					}
					if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
						p.Name,
						p.Version,
						severity,
						p.Author); err != nil {
						log.Debug().Err(err).Msg("Failed to write plugin entry")
					}
				}
			} else {
				fmt.Fprintln(w, "NAME\tVERSION\tSEVERITY")
				fmt.Fprintln(w, "----\t-------\t--------")
				for _, p := range pluginsToDisplay {
					severity := string(p.Metadata.Severity)
					if severity == "" {
						severity = "info"
					}
					if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n",
						p.Name,
						p.Version,
						severity); err != nil {
						log.Debug().Err(err).Msg("Failed to write plugin entry")
					}
				}
			}
			if err := w.Flush(); err != nil {
				log.Warn().Err(err).Msg("Failed to flush output")
			}

			// Print category summary if showing all plugins
			if category == "" && !verbose {
				fmt.Println()
				fmt.Println("Categories:")
				categoryCount := make(map[plugin.Category]int)
				for cat, catPlugins := range plugins {
					categoryCount[cat] = len(catPlugins)
				}

				// Sort categories for consistent output
				var categories []plugin.Category
				for cat := range categoryCount {
					categories = append(categories, cat)
				}
				sort.Slice(categories, func(i, j int) bool {
					return categories[i].String() < categories[j].String()
				})

				for _, cat := range categories {
					fmt.Printf("  %s: %d plugins\n", cat.String(), categoryCount[cat])
				}

				fmt.Println()
				fmt.Println("Use --category flag to filter by category.")
				fmt.Println("Use --verbose flag to show detailed information.")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed information")
	cmd.Flags().StringVar(&category, "category", "", "Filter by category (ssh, http, tls, database, network, misc)")

	return cmd
}
