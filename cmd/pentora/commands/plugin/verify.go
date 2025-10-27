package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/spf13/cobra"
)

func newVerifyCommand() *cobra.Command {
	var cacheDir string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify plugin checksums",
		Long: `Verify the integrity of installed plugins by checking their SHA-256 checksums.

By default, verifies all installed plugins. Use --plugin to verify a specific plugin.

Exit codes:
  0 - All plugins verified successfully
  1 - One or more plugins failed verification or error occurred`,
		Example: `  # Verify all installed plugins
  pentora plugin verify

  # Verify a specific plugin
  pentora plugin verify --plugin ssh-cve-2024-6387

  # Verify plugins in custom cache directory
  pentora plugin verify --cache-dir /custom/path

  # JSON output
  pentora plugin verify --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			opts, err := bind.BindVerifyOptions(cmd)
			if err != nil {
				return err
			}

			// Call service layer
			result, err := svc.Verify(cmd.Context(), opts)
			if err != nil {
				return formatter.PrintError(err)
			}

			// Print results
			if err := printVerifyResult(formatter, result); err != nil {
				return err
			}

			// Return error if any plugins failed
			if result.FailedCount > 0 {
				return fmt.Errorf("verification failed for %d plugin(s)", result.FailedCount)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: platform-specific, see storage config)")
	cmd.Flags().String("plugin", "", "Verify specific plugin by name")
	cmd.Flags().String("output", "table", "Output format: json, table")
	cmd.Flags().Bool("quiet", false, "Suppress non-essential output")
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

// printVerifyResult formats and prints the verify result using the formatter
func printVerifyResult(f format.Formatter, result *plugin.VerifyResult) error {
	if result.TotalCount == 0 {
		return f.PrintSummary("No plugins installed to verify.")
	}

	// Print header
	if err := f.PrintSummary(fmt.Sprintf("Verifying %d plugin(s)...", result.TotalCount)); err != nil {
		return err
	}

	// Build table rows
	var rows [][]string
	for _, r := range result.Results {
		status := "✓ OK"
		if !r.Valid {
			switch r.ErrorType {
			case "missing":
				status = "✗ File not found"
			case "checksum":
				status = "✗ Checksum mismatch"
			case "error":
				status = fmt.Sprintf("✗ Error: %v", r.Error)
			default:
				status = "✗ Failed"
			}
		}
		rows = append(rows, []string{r.ID, r.Version, status})
	}

	// Print table
	if err := f.PrintTable([]string{"Plugin", "Version", "Status"}, rows); err != nil {
		return err
	}

	// Print summary
	if result.FailedCount == 0 {
		return f.PrintSummary(fmt.Sprintf("✓ All %d plugin(s) verified successfully", result.TotalCount))
	}

	return f.PrintSummary(fmt.Sprintf("✗ %d plugin(s) failed verification", result.FailedCount))
}
