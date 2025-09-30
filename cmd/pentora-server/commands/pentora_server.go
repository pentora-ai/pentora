package commands

import (
	"fmt"
	"os"

	"github.com/pentora-ai/pentora/pkg/cli"
	"github.com/pentora-ai/pentora/pkg/workspace"
	"github.com/spf13/cobra"
)

const cliExecutable = "pentora-server"

func NewCommand() *cobra.Command {
	var workspaceDir string

	command := &cobra.Command{
		Use:   cliExecutable,
		Short: "Pentora API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := workspace.Prepare(workspaceDir)
			return err
		},
	}

	command.AddCommand(cli.NewVersionCommand(cliExecutable))
	command.Flags().StringVar(&workspaceDir, "workspace-dir", "", "Workspace root directory (defaults to OS-specific path)")
	command.SilenceUsage = true
	command.SilenceErrors = true

	command.SuggestionsMinimumDistance = 1

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}

	return command
}
