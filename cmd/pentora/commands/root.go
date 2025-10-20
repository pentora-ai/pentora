package commands

import (
	"context"
	"fmt"

	serverCmd "github.com/pentora-ai/pentora/cmd/pentora/commands/server"
	"github.com/pentora-ai/pentora/pkg/appctx"
	"github.com/pentora-ai/pentora/pkg/cli"
	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/workspace"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const cliExecutable = "pentora"

// NewCommand constructs the top-level pentora CLI command, wiring global flags,
// AppManager lifecycle, and shared workspace preparation.
func NewCommand() *cobra.Command {
	var (
		configFile        string
		workspaceDir      string
		workspaceDisabled bool
		appManager        engine.Manager
		verbosityCount    int
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

			if !workspaceDisabled {
				prepared, err := workspace.Prepare(workspaceDir)
				if err != nil {
					return fmt.Errorf("prepare workspace: %w", err)
				}
				ctx = workspace.WithContext(ctx, prepared)
				log.Info().Str("workspace", prepared).Msg("workspace ready")
			} else {
				log.Info().Msg("workspace disabled for this run")
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
	cmd.PersistentFlags().StringVar(&workspaceDir, "workspace-dir", "", "Override workspace root directory")
	cmd.PersistentFlags().BoolVar(&workspaceDisabled, "no-workspace", false, "Disable workspace persistence for this run")
	cmd.PersistentFlags().CountVarP(&verbosityCount, "verbosity", "v", "Increase logging verbosity (repeatable)")

	config.BindFlags(cmd.PersistentFlags())

	cmd.AddGroup(&cobra.Group{ID: "scan", Title: "Scan Commands"})
	cmd.AddGroup(&cobra.Group{ID: "core", Title: "Core Commands"})

	cmd.AddCommand(serverCmd.NewCommand())
	cmd.AddCommand(cli.DiscoverCmd)
	cmd.AddCommand(cli.ServeCmd)
	cmd.AddCommand(cli.NewVersionCommand(cliExecutable))
	cmd.AddCommand(ScanCmd)
	cmd.AddCommand(NewFingerprintCommand())

	return cmd
}
