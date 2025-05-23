// cmd/pentora/pentora.go
package main

import (
	"fmt"
	"os"

	"github.com/pentora-ai/pentora/cmd/version"
	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/spf13/cobra"

	"github.com/rs/zerolog/log"
)

var (
	// rootCmd is the base command when no subcommands are given.
	rootCmd = &cobra.Command{
		Use:   "pentora",
		Short: "Pentora - Platform-independent vulnerability scanner",
		Long:  `Pentora is a modern, cross-platform security scanner designed to efficiently find vulnerabilities and misconfigurations in your infrastructure. It supports modular scanning, extensible plugins, and various output formats.`,
		// PersistentPreRunE is run before any subcommand's RunE.
		// This is a good place to initialize shared components like AppManager.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error

			// Create an instance of your AppManagerFactory
			// This factory could be more complex or configurable itself if needed.
			factory := engine.DefaultAppManagerFactory{}

			// Use the factory to create and initialize the AppManager
			// The factory will handle loading the configuration using flags and configFile.
			appManager, err = factory.Create(cmd.Flags(), configFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating application manager via factory: %v\n", err)
				return err
			}

			log.Debug().Msgf("Pentora is running: %v", appManager.Version.Version)

			// InitializeGlobalLogger(appManager.ConfigManager.Get().Log) // If you have one
			log.Printf("AppManager created and initialized via factory in PersistentPreRunE. Config Log Level: %s", appManager.ConfigManager.Get().Log.Level)
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Default action when 'pentora' is run without subcommands.
			// log.Debug().Msg("Pentora Security Scanner. Use 'pentora --help' for a list of commands.")
			if appManager != nil && appManager.ConfigManager != nil {
				// This demonstrates accessing the loaded config.
				currentCfg := appManager.ConfigManager.Get()
				log.Debug().Msgf("Hint: Current log level from config is '%s'.\n", currentCfg.Log.Level)
			}

			//spew.Dump(appManager) // This is a debugging line to print the appManager struct.
		},
	}

	// configFile is a package-level variable to store the path to the configuration file,
	// set by a persistent flag.
	configFile string

	// appManager is a package-level variable holding the initialized application manager.
	// It's populated in PersistentPreRunE and can be accessed by subcommands.
	appManager *engine.AppManager
)

// init is called by Go when the package is initialized.
// It's used here to define flags and add subcommands to the root command.
func init() {
	// Define a persistent flag for the configuration file path.
	// This flag will be available to the root command and all its subcommands.
	// The default value is empty, meaning config.Load will try default locations.
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Configuration file path (e.g., ./config.yaml, /etc/pentora/config.yaml)")

	// Define other application-wide persistent flags that correspond to config settings.
	// These flags will be bound to Koanf by the config.Manager.
	// Using a helper function from the config package to define these flags
	// ensures consistency with the Config struct.
	config.BindFlags(rootCmd.PersistentFlags()) // This binds flags based on DefaultConfig

	// Add subcommands to the root command.
	// These subcommands (ScanCmd, ServeCmd, etc.) might use the initialized appManager.
	// rootCmd.AddCommand(cli.ScanCmd)        // Assumes cli.ScanCmd is adapted to use appManager if needed
	// rootCmd.AddCommand(cli.ServeCmd)       // Assumes cli.ServeCmd uses appManager for config, logger, etc.
	// rootCmd.AddCommand(license.LicenseCmd) // License commands
	rootCmd.AddCommand(version.VersionCmd) // Version command
	// rootCmd.AddCommand(cli.ModuleHostCmd)  // The test command for hosting external modules
}

// main is the entry point of the Pentora application.
func main() {
	// Execute the root Cobra command. Cobra handles parsing flags and
	// routing to the appropriate subcommand.
	// AppManager initialization happens in rootCmd.PersistentPreRunE.
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints errors by default, but we can log it explicitly if needed.
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1) // Exit with a non-zero code to indicate failure.
	}
}
