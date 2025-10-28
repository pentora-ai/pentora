package plugin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
)

// getFormatter creates a formatter from command flags
func getFormatter(cmd *cobra.Command) format.Formatter {
	outputMode := format.ParseMode(cmd.Flag("output").Value.String())
	quiet, _ := cmd.Flags().GetBool("quiet")
	noColor, _ := cmd.Flags().GetBool("no-color")
	return format.New(os.Stdout, os.Stderr, outputMode, quiet, !noColor)
}

// getPluginService creates a plugin service with the given cache directory
// If cacheDir is empty, uses the default platform-specific cache directory
func getPluginService(cacheDir string) (*plugin.Service, error) {
	if cacheDir == "" {
		storageConfig, err := storage.DefaultConfig()
		if err != nil {
			return nil, fmt.Errorf("get storage config: %w", err)
		}
		cacheDir = filepath.Join(storageConfig.WorkspaceRoot, "plugins", "cache")
	}

	svc, err := plugin.NewService(plugin.WithCacheDir(cacheDir))
	if err != nil {
		return nil, fmt.Errorf("create plugin service: %w", err)
	}
	return svc, nil
}

// handlePartialFailure handles partial failure errors by printing results and exiting with code 8
func handlePartialFailure(err error, formatter format.Formatter, printFunc func() error) error {
	if err != nil && errors.Is(err, plugin.ErrPartialFailure) {
		// Print result even on partial failure
		if printErr := printFunc(); printErr != nil {
			return printErr
		}
		// Exit with code 8 for partial failure
		os.Exit(plugin.ExitCode(err))
	}
	return nil
}

// convertPluginErrors converts plugin errors to format.ErrorDetail
func convertPluginErrors(errors []plugin.PluginError) []format.ErrorDetail {
	errorDetails := make([]format.ErrorDetail, 0, len(errors))
	for _, e := range errors {
		errorDetails = append(errorDetails, format.ErrorDetail{
			PluginID:  e.PluginID,
			Error:     e.Error,
			ErrorCode: e.Code,
		})
	}
	return errorDetails
}

// buildPluginTable builds table rows for plugin list
// Used by install and update commands
func buildPluginTable(plugins []*plugin.PluginInfo) [][]string {
	var rows [][]string
	for _, p := range plugins {
		categoryStr := ""
		if len(p.Tags) > 0 {
			categoryStr = p.Tags[0]
		}
		rows = append(rows, []string{p.Name, p.Version, categoryStr})
	}
	return rows
}

// formatBytes formats bytes as human-readable string (e.g., "1.5 MiB")
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
