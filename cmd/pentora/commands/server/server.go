package server

import (
	"github.com/pentora-ai/pentora/pkg/workspace"
	"github.com/spf13/cobra"
)

const cliExecutable = "server"

func NewCommand() *cobra.Command {
	var workspaceDir string

	command := &cobra.Command{
		Use:   cliExecutable,
		Short: "Pentora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := workspace.Prepare(workspaceDir)
			if err != nil {
				return err
			}

			return cmd.Help()
		},
	}

	command.Flags().StringVar(&workspaceDir, "workspace-dir", "", "Workspace root directory (defaults to OS-specific path)")

	command.SuggestionsMinimumDistance = 1

	// Subcommands
	command.AddCommand(newStartServerCommand())

	return command
}
