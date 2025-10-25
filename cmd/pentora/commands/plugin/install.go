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

			// Create service
			svc, err := plugin.NewService(cacheDir)
			if err != nil {
				return fmt.Errorf("create plugin service: %w", err)
			}

			// Build install options
			opts := plugin.InstallOptions{
				Force: force,
			}

			if source != "" {
				opts.Source = source
			}

			// Call service layer
			result, err := svc.Install(ctx, target, opts)
			if err != nil {
				return err
			}

			// Print results
			printInstallResult(result)

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")
	cmd.Flags().StringVar(&source, "source", "", "Install from specific source (e.g., 'official')")
	cmd.Flags().BoolVar(&force, "force", false, "Force re-install even if already cached")

	return cmd
}

// printInstallResult formats and prints the install result
func printInstallResult(result *plugin.InstallResult) {
	// Show plugins that were processed
	if len(result.Plugins) > 0 {
		fmt.Println("\nProcessed plugins:")
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
		fmt.Println()
	}

	// Summary
	fmt.Printf("Installation Summary:\n")
	fmt.Printf("  Installed: %d\n", result.InstalledCount)
	if result.SkippedCount > 0 {
		fmt.Printf("  Already installed: %d\n", result.SkippedCount)
	}
	if result.FailedCount > 0 {
		fmt.Printf("  Failed: %d\n", result.FailedCount)
	}

	if result.InstalledCount > 0 {
		fmt.Printf("\n✓ Plugins installed successfully\n")
	}
}
