package main

import (
	"os"
	"path/filepath"

	pentoraServerCli "github.com/pentora-ai/pentora/cmd/pentora-server/commands"
	pentoraCli "github.com/pentora-ai/pentora/cmd/pentora/commands"
	"github.com/spf13/cobra"
)

const cliExecutableEnv = "PENTORA_CLI_EXECUTABLE"

// main initializes the CLI application by determining the executable name and selecting
// the appropriate command to execute. It checks the executable name from the environment
// or the current process, then switches between available CLI commands (pentora or pentora-server).
// If the command execution fails, the application exits with a non-zero status code.
func main() {
	var command *cobra.Command

	cliExecutable := filepath.Base(os.Args[0])
	if val := os.Getenv(cliExecutableEnv); val != "" {
		cliExecutable = val
	}

	switch cliExecutable {
	case "pentora":
		command = pentoraCli.NewCommand()
	case "pentora-server":
		command = pentoraServerCli.NewCommand()
	default:
		command = pentoraCli.NewCommand()
	}

	err := command.Execute()
	if err != nil {
		os.Exit(1) 
	}
}
