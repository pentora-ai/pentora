package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/spf13/cobra"
)

func newInstallCommand() *cobra.Command {
	var cacheDir string

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
  pentora plugin install ssh --force

  # JSON output
  pentora plugin install ssh --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			// Create formatter
			outputMode := format.ParseMode(cmd.Flag("output").Value.String())
			quiet, _ := cmd.Flags().GetBool("quiet")
			noColor, _ := cmd.Flags().GetBool("no-color")
			formatter := format.New(os.Stdout, os.Stderr, outputMode, quiet, !noColor)

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
			opts, err := bind.BindInstallOptions(cmd)
			if err != nil {
				return err
			}

			// Call service layer
			result, err := svc.Install(ctx, target, opts)
			if err != nil {
				return formatter.PrintError(err)
			}

			// Print results using formatter
			return printInstallResult(formatter, result)
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().String("source", "", "Install from specific source (e.g., 'official')")
	cmd.Flags().Bool("force", false, "Force re-install even if already cached")
	cmd.Flags().String("output", "table", "Output format: json, table")
	cmd.Flags().Bool("quiet", false, "Suppress non-essential output")
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

// printInstallResult formats and prints the install result using the formatter
func printInstallResult(f format.Formatter, result *plugin.InstallResult) error {
	// JSON mode: output complete result as JSON
	if f == nil {
		return fmt.Errorf("formatter is nil")
	}

	// Build table rows
	var rows [][]string
	for _, p := range result.Plugins {
		categoryStr := ""
		if len(p.Tags) > 0 {
			categoryStr = p.Tags[0]
		}
		rows = append(rows, []string{p.Name, p.Version, categoryStr})
	}

	// Print table if there are plugins
	if len(rows) > 0 {
		if err := f.PrintTable([]string{"Name", "Version", "Category"}, rows); err != nil {
			return err
		}
	}

	// Build summary message
	summary := fmt.Sprintf("Installation Summary: Installed: %d", result.InstalledCount)
	if result.SkippedCount > 0 {
		summary += fmt.Sprintf(", Already installed: %d", result.SkippedCount)
	}
	if result.FailedCount > 0 {
		summary += fmt.Sprintf(", Failed: %d", result.FailedCount)
	}

	// Print summary
	if err := f.PrintSummary(summary); err != nil {
		return err
	}

	// Success message
	if result.InstalledCount > 0 {
		return f.PrintSummary("✓ Plugins installed successfully")
	}

	return nil
}
