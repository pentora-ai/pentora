// Package storage provides CLI commands for managing Vulntor storage.
package storage

import (
	"github.com/spf13/cobra"
)

// NewStorageCommand creates and returns the 'vulntor storage' command.
//
// This command provides subcommands for storage management operations:
//   - gc: Garbage collection to clean up old scans
//
// Example usage:
//
//	vulntor storage gc
//	vulntor storage gc --dry-run
//	vulntor storage gc --max-scans=100
func NewStorageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Manage Vulntor storage",
		Long: `Manage Vulntor storage operations including garbage collection.

The storage command provides utilities for managing scan data persistence,
retention policies, and cleanup operations.`,
	}

	// Add subcommands
	cmd.AddCommand(newGCCommand())

	return cmd
}
