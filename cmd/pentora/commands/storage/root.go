// Package storage provides CLI commands for managing Pentora storage.
package storage

import (
	"github.com/spf13/cobra"
)

// NewStorageCommand creates and returns the 'pentora storage' command.
//
// This command provides subcommands for storage management operations:
//   - gc: Garbage collection to clean up old scans
//
// Example usage:
//
//	pentora storage gc
//	pentora storage gc --dry-run
//	pentora storage gc --max-scans=100
func NewStorageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Manage Pentora storage",
		Long: `Manage Pentora storage operations including garbage collection.

The storage command provides utilities for managing scan data persistence,
retention policies, and cleanup operations.`,
	}

	// Add subcommands
	cmd.AddCommand(newGCCommand())

	return cmd
}
