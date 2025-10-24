package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newUninstallCommand() *cobra.Command {
	var (
		cacheDir string
		all      bool
		category string
	)

	cmd := &cobra.Command{
		Use:     "uninstall <plugin-name>",
		Aliases: []string{"remove", "rm"},
		Short:   "Uninstall plugins",
		Long: `Uninstall (remove) plugins from the local cache.

This command removes plugins from the cache directory. You can uninstall specific plugins by name,
all plugins in a category, or all plugins at once.`,
		Example: `  # Uninstall specific plugin
  pentora plugin uninstall ssh-cve-2024-6387

  # Uninstall all SSH plugins
  pentora plugin uninstall --category ssh

  # Uninstall all plugins
  pentora plugin uninstall --all

  # Alternative commands (aliases)
  pentora plugin remove ssh-cve-2024-6387
  pentora plugin rm ssh-cve-2024-6387`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate args
			if !all && category == "" && len(args) == 0 {
				return fmt.Errorf("must specify plugin name, --category, or --all")
			}

			if all && category != "" {
				return fmt.Errorf("cannot use --all and --category together")
			}

			if all && len(args) > 0 {
				return fmt.Errorf("cannot specify plugin name with --all")
			}

			if category != "" && len(args) > 0 {
				return fmt.Errorf("cannot specify plugin name with --category")
			}

			// Use default cache dir if not specified
			if cacheDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("get home directory: %w", err)
				}
				cacheDir = filepath.Join(homeDir, ".pentora", "plugins", "cache")
			}

			// Create manifest manager
			manifestPath := filepath.Join(filepath.Dir(cacheDir), "registry.json")
			manifestMgr, err := plugin.NewManifestManager(manifestPath)
			if err != nil {
				return fmt.Errorf("create manifest manager: %w", err)
			}

			// Get list of installed plugins
			entries, err := manifestMgr.List()
			if err != nil {
				return fmt.Errorf("list plugins: %w", err)
			}

			if len(entries) == 0 {
				fmt.Println("No plugins installed.")
				return nil
			}

			// Determine which plugins to uninstall
			var toUninstall []*plugin.ManifestEntry

			if all {
				// Uninstall all plugins
				toUninstall = entries
				fmt.Printf("Uninstalling all %d plugin(s)...\n", len(entries))
			} else if category != "" {
				// Uninstall by category
				for _, entry := range entries {
					for _, tag := range entry.Tags {
						if tag == category {
							toUninstall = append(toUninstall, entry)
							break
						}
					}
				}
				if len(toUninstall) == 0 {
					return fmt.Errorf("no plugins found in category '%s'", category)
				}
				fmt.Printf("Uninstalling %d plugin(s) from category '%s'...\n", len(toUninstall), category)
			} else {
				// Uninstall specific plugin by name or ID
				pluginName := args[0]
				pluginNameLower := strings.ToLower(pluginName)
				found := false
				for _, entry := range entries {
					// Match by name or generated ID
					entryID := plugin.GeneratePluginID(entry.Name)
					if entry.Name == pluginName || entryID == pluginNameLower {
						toUninstall = append(toUninstall, entry)
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("plugin '%s' not found (not installed)", pluginName)
				}
				fmt.Printf("Uninstalling plugin '%s'...\n", pluginName)
			}

			// Uninstall plugins
			removedCount := 0
			failedCount := 0

			for _, entry := range toUninstall {
				fmt.Printf("  Removing %s v%s...", entry.Name, entry.Version)

				// Remove plugin file
				pluginPath := filepath.Join(cacheDir, entry.Path)
				if err := os.Remove(pluginPath); err != nil {
					if !os.IsNotExist(err) {
						fmt.Printf(" ✗ (failed to remove file: %v)\n", err)
						log.Warn().
							Str("plugin", entry.Name).
							Err(err).
							Msg("Failed to remove plugin file")
						failedCount++
						continue
					}
					// File doesn't exist, but still remove from manifest
				}

				// Remove from manifest (use ID)
				if err := manifestMgr.Remove(entry.ID); err != nil {
					fmt.Printf(" ✗ (failed to update manifest: %v)\n", err)
					log.Warn().
						Str("plugin", entry.Name).
						Err(err).
						Msg("Failed to remove plugin from manifest")
					failedCount++
					continue
				}

				fmt.Printf(" ✓\n")
				removedCount++
			}

			// Save manifest to disk if any plugins were removed
			if removedCount > 0 {
				if err := manifestMgr.Save(); err != nil {
					log.Warn().Err(err).Msg("Failed to save plugin manifest")
					fmt.Printf("\nWarning: Failed to update plugin registry\n")
				}
			}

			// Summary
			fmt.Printf("\nUninstall Summary:\n")
			fmt.Printf("  Removed: %d\n", removedCount)
			if failedCount > 0 {
				fmt.Printf("  Failed: %d\n", failedCount)
			}

			remaining := len(entries) - removedCount
			if remaining > 0 {
				fmt.Printf("  Remaining in cache: %d\n", remaining)
			} else {
				fmt.Println("\n✓ All plugins uninstalled. Cache is empty.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Plugin cache directory (default: ~/.pentora/plugins/cache)")
	cmd.Flags().BoolVar(&all, "all", false, "Uninstall all plugins")
	cmd.Flags().StringVar(&category, "category", "", "Uninstall all plugins from category (ssh, http, tls, database, network)")

	return cmd
}
