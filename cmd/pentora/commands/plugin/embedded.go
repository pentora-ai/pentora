package plugin

import (
	"fmt"
	"os"
	"sort"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
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
  pentora plugin embedded --category tls --verbose

  # JSON output
  pentora plugin embedded --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create formatter
			outputMode := format.ParseMode(cmd.Flag("output").Value.String())
			quiet, _ := cmd.Flags().GetBool("quiet")
			noColor, _ := cmd.Flags().GetBool("no-color")
			formatter := format.New(os.Stdout, os.Stderr, outputMode, quiet, !noColor)

			// Load embedded plugins
			plugins, err := plugin.LoadEmbeddedPlugins()
			if err != nil {
				return formatter.PrintError(fmt.Errorf("load embedded plugins: %w", err))
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
					return formatter.PrintError(fmt.Errorf("unknown category: %s (valid: ssh, http, tls, database, network, misc)", category))
				}

				// Get plugins for the specified category
				if catPlugins, ok := plugins[categoryFilter]; ok {
					pluginsToDisplay = catPlugins
				} else {
					return formatter.PrintSummary(fmt.Sprintf("No embedded plugins found for category: %s", category))
				}
			} else {
				// Flatten all plugins
				for _, catPlugins := range plugins {
					pluginsToDisplay = append(pluginsToDisplay, catPlugins...)
				}
			}

			if len(pluginsToDisplay) == 0 {
				return formatter.PrintSummary("No embedded plugins found.")
			}

			// Sort plugins by name for consistent output
			sort.Slice(pluginsToDisplay, func(i, j int) bool {
				return pluginsToDisplay[i].Name < pluginsToDisplay[j].Name
			})

			// Print header
			var headerMsg string
			if category != "" {
				headerMsg = fmt.Sprintf("Found %d embedded plugin(s) in category '%s'", len(pluginsToDisplay), category)
			} else {
				headerMsg = fmt.Sprintf("Found %d embedded plugin(s) total", len(pluginsToDisplay))
			}
			if err := formatter.PrintSummary(headerMsg); err != nil {
				return err
			}

			// Build table
			var headers []string
			var rows [][]string

			if verbose {
				headers = []string{"Name", "Version", "Severity", "Author"}
				for _, p := range pluginsToDisplay {
					severity := string(p.Metadata.Severity)
					if severity == "" {
						severity = "info"
					}
					rows = append(rows, []string{p.Name, p.Version, severity, p.Author})
				}
			} else {
				headers = []string{"Name", "Version", "Severity"}
				for _, p := range pluginsToDisplay {
					severity := string(p.Metadata.Severity)
					if severity == "" {
						severity = "info"
					}
					rows = append(rows, []string{p.Name, p.Version, severity})
				}
			}

			// Print table
			if err := formatter.PrintTable(headers, rows); err != nil {
				return err
			}

			// Print category summary if showing all plugins (non-verbose, non-quiet)
			if category == "" && !verbose {
				// Build category count
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

				// Print category breakdown
				if err := formatter.PrintSummary(""); err != nil {
					return err
				}
				if err := formatter.PrintSummary("Categories:"); err != nil {
					return err
				}
				for _, cat := range categories {
					if err := formatter.PrintSummary(fmt.Sprintf("  %s: %d plugins", cat.String(), categoryCount[cat])); err != nil {
						return err
					}
				}

				if err := formatter.PrintSummary(""); err != nil {
					return err
				}
				if err := formatter.PrintSummary("Use --category flag to filter by category."); err != nil {
					return err
				}
				if err := formatter.PrintSummary("Use --verbose flag to show detailed information."); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed information")
	cmd.Flags().StringVar(&category, "category", "", "Filter by category (ssh, http, tls, database, network, misc)")
	cmd.Flags().String("output", "table", "Output format: json, table")
	cmd.Flags().Bool("quiet", false, "Suppress non-essential output")
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}
