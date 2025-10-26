package main

import (
	"os"

	pentoraCli "github.com/pentora-ai/pentora/cmd/pentora/commands"
)

// main initializes the CLI application by determining the executable name and selecting
// the appropriate command to execute. It checks the executable name from the environment
// or the current process, then switches between available CLI commands (pentora or pentora-server).
// If the command execution fails, the application exits with a non-zero status code.
func main() {
	command := pentoraCli.NewCommand()

	err := command.Execute()
	if err != nil {
		os.Exit(1)
	}
}
