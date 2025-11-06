package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	dagCmd "github.com/pentora-ai/pentora/cmd/pentora/commands/dag"
	pluginCmd "github.com/pentora-ai/pentora/cmd/pentora/commands/plugin"
	serverCmd "github.com/pentora-ai/pentora/cmd/pentora/commands/server"
	storageCmd "github.com/pentora-ai/pentora/cmd/pentora/commands/storage"
	"github.com/pentora-ai/pentora/pkg/appctx"
	"github.com/pentora-ai/pentora/pkg/cli"
	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/engine"
	// Register all available modules for DAG execution
	_ "github.com/pentora-ai/pentora/pkg/modules/evaluation" // Vulnerability evaluation modules
	_ "github.com/pentora-ai/pentora/pkg/modules/parse"      // Protocol parser modules
	_ "github.com/pentora-ai/pentora/pkg/modules/reporting"  // Reporting modules
	_ "github.com/pentora-ai/pentora/pkg/modules/scan"       // Scanner modules
	"github.com/pentora-ai/pentora/pkg/storage"
)

const cliExecutable = "pentora"

// NewCommand constructs the top-level pentora CLI command, wiring global flags,
// AppManager lifecycle, and shared workspace preparation.
func NewCommand() *cobra.Command {
	var (
		configFile      string
		storageDir      string
		storageDisabled bool
		appManager      engine.Manager
		verbosityCount  int
		verbose         bool
	)

	cmd := &cobra.Command{
		Use:   cliExecutable,
		Short: "Pentora is a fast and flexible network scanner",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			factory := &engine.DefaultAppManagerFactory{}

			mgr, err := factory.Create(cmd.Flags(), configFile)
			if err != nil {
				return fmt.Errorf("initialize AppManager: %w", err)
			}
			appManager = mgr

			ctx := context.WithValue(cmd.Context(), engine.AppManagerKey, appManager)
			ctx = appctx.WithConfig(ctx, appManager.Config())

			if !storageDisabled {
				storageConfig, err := storage.DefaultConfig()
				if err != nil {
					return fmt.Errorf("get storage config: %w", err)
				}
				if storageDir != "" {
					storageConfig.WorkspaceRoot = storageDir
				}
				ctx = storage.WithConfig(ctx, storageConfig)
				log.Info().Str("storage_root", storageConfig.WorkspaceRoot).Msg("storage ready")
			} else {
				log.Info().Msg("storage disabled for this run")
			}

			// Configure global log level based on verbosity flags
			// If explicit --verbose is set, show debug and above
			// Else use -v count: 0=>Error, 1=>Info, 2+=>Debug
			if verbose {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			} else {
				switch {
				case verbosityCount <= 0:
					zerolog.SetGlobalLevel(zerolog.ErrorLevel)
				case verbosityCount == 1:
					zerolog.SetGlobalLevel(zerolog.InfoLevel)
				default:
					zerolog.SetGlobalLevel(zerolog.DebugLevel)
				}
			}

			cmd.SetContext(ctx)
			if root := cmd.Root(); root != nil && root != cmd {
				root.SetContext(ctx)
			}
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if appManager != nil {
				appManager.Shutdown()
			}
			return nil
		},
	}

	cmd.SilenceUsage = true

	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	cmd.PersistentFlags().StringVar(&storageDir, "storage-dir", "", "Override storage root directory")
	cmd.PersistentFlags().BoolVar(&storageDisabled, "no-storage", false, "Disable storage persistence for this run")
	cmd.PersistentFlags().CountVarP(&verbosityCount, "verbosity", "v", "Increase logging verbosity (repeatable)")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging (shows service layer logs)")

	config.BindFlags(cmd.PersistentFlags())

	cmd.AddGroup(&cobra.Group{ID: "scan", Title: "Scan Commands"})
	cmd.AddGroup(&cobra.Group{ID: "core", Title: "Core Commands"})

	cmd.AddCommand(dagCmd.NewCommand())
	cmd.AddCommand(pluginCmd.NewCommand())
	cmd.AddCommand(serverCmd.NewCommand())
	cmd.AddCommand(storageCmd.NewStorageCommand())
	cmd.AddCommand(cli.NewVersionCommand(cliExecutable))
	cmd.AddCommand(ScanCmd)
	cmd.AddCommand(NewFingerprintCommand())
	cmd.AddCommand(NewStatsCommand())

	return cmd
}
