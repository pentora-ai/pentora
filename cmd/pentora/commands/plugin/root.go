// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the plugin command with all subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Plugin management",
		Long: `Manage YAML-based vulnerability detection plugins.

Plugins extend Pentora's vulnerability detection capabilities with custom rules.
Use these commands to list, inspect, verify, and maintain your plugin cache.`,
		Example: `  # List all installed plugins
  pentora plugin list

  # Show details for a specific plugin
  pentora plugin info ssh-cve-2024-6387

  # Verify plugin checksums
  pentora plugin verify

  # Clean unused cache entries
  pentora plugin clean`,
	}

	// Add subcommands
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newEmbeddedCommand())
	cmd.AddCommand(newInfoCommand())
	cmd.AddCommand(newVerifyCommand())
	cmd.AddCommand(newCleanCommand())

	return cmd
}
