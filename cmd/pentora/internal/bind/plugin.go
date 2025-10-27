// Package bind provides centralized flag-to-options binding for CLI commands.
//
// This package implements the binding layer between Cobra command flags and
// service layer DTOs (Data Transfer Objects). By centralizing this logic,
// we ensure consistency, testability, and maintainability across all commands.
package bind

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/pentora-ai/pentora/pkg/plugin"
)

// BindInstallOptions extracts and validates InstallOptions from command flags.
//
// This function reads the install-specific flags from the Cobra command and
// constructs a properly validated InstallOptions struct for the service layer.
//
// Flags read:
//   - --source: Optional plugin source name
//   - --force: Force re-install flag
//
// Returns an error if validation fails (though currently install has no required flags).
func BindInstallOptions(cmd *cobra.Command) (plugin.InstallOptions, error) {
	source, _ := cmd.Flags().GetString("source")
	force, _ := cmd.Flags().GetBool("force")

	opts := plugin.InstallOptions{
		Source: source,
		Force:  force,
	}

	return opts, nil
}

// BindUpdateOptions extracts and validates UpdateOptions from command flags.
//
// This function reads the update-specific flags from the Cobra command and
// constructs a properly validated UpdateOptions struct for the service layer.
//
// Flags read:
//   - --category: Optional category filter (ssh, http, tls, database, network, misc)
//   - --source: Optional plugin source name
//   - --force: Force re-download flag
//   - --dry-run: Dry run mode (preview only)
//
// Returns an error if validation fails (e.g., invalid category).
func BindUpdateOptions(cmd *cobra.Command) (plugin.UpdateOptions, error) {
	category, _ := cmd.Flags().GetString("category")
	source, _ := cmd.Flags().GetString("source")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	opts := plugin.UpdateOptions{
		Source: source,
		Force:  force,
		DryRun: dryRun,
	}

	// Convert category string to Category type (validation happens in service layer)
	if category != "" {
		opts.Category = plugin.Category(category)
	}

	return opts, nil
}

// BindUninstallOptions extracts and validates UninstallOptions from command flags.
//
// This function reads the uninstall-specific flags from the Cobra command and
// constructs a properly validated UninstallOptions struct for the service layer.
//
// Flags read:
//   - --all: Uninstall all plugins flag
//   - --category: Optional category filter (ssh, http, tls, database, network, misc)
//
// Returns an error if validation fails or if conflicting flags are provided.
func BindUninstallOptions(cmd *cobra.Command) (plugin.UninstallOptions, error) {
	all, _ := cmd.Flags().GetBool("all")
	category, _ := cmd.Flags().GetString("category")

	opts := plugin.UninstallOptions{
		All: all,
	}

	// Convert category string to Category type (validation happens in service layer)
	if category != "" {
		opts.Category = plugin.Category(category)
	}

	// Validate flag conflicts
	if all && category != "" {
		return opts, fmt.Errorf("cannot use --all and --category together")
	}

	return opts, nil
}

// BindCleanOptions extracts and validates CleanOptions from command flags.
//
// This function reads the clean-specific flags from the Cobra command and
// constructs a properly validated CleanOptions struct for the service layer.
//
// Flags read:
//   - --older-than: Duration string for removing old cache entries (e.g., "720h" for 30 days)
//   - --dry-run: Dry run mode (preview only)
//
// Returns an error if the duration string is invalid.
func BindCleanOptions(cmd *cobra.Command) (plugin.CleanOptions, error) {
	olderThan, _ := cmd.Flags().GetString("older-than")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Parse duration
	duration, err := time.ParseDuration(olderThan)
	if err != nil {
		return plugin.CleanOptions{}, fmt.Errorf("invalid duration '%s': %w (use format like '720h' for 30 days)", olderThan, err)
	}

	opts := plugin.CleanOptions{
		OlderThan: duration,
		DryRun:    dryRun,
	}

	return opts, nil
}

// BindVerifyOptions extracts and validates VerifyOptions from command flags.
//
// This function reads the verify-specific flags from the Cobra command and
// constructs a properly validated VerifyOptions struct for the service layer.
//
// Flags read:
//   - --plugin: Optional plugin name to verify (if empty, verifies all plugins)
//
// Returns an error if validation fails.
func BindVerifyOptions(cmd *cobra.Command) (plugin.VerifyOptions, error) {
	pluginName, _ := cmd.Flags().GetString("plugin")

	opts := plugin.VerifyOptions{
		PluginID: pluginName,
	}

	return opts, nil
}
