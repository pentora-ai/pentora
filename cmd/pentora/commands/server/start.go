// Package server provides the Cobra command implementation for the Pentora server lifecycle.
// It wires CLI flags to the server runtime and handles the start/stop commands.
package server

import (
	"github.com/pentora-ai/pentora/pkg/server"
	"github.com/spf13/cobra"
)

// newStartServerCommand creates and returns the 'pentora server start' command.
//
// This command initializes the Pentora server runtime, which includes:
//   - HTTP API server with REST endpoints (/api/v1/scans, etc.)
//   - Static UI asset serving (/ui/*)
//   - Health and readiness endpoints (/healthz, /readyz)
//   - Background job workers (scan execution, scheduling, notifications)
//
// The server runs until interrupted (SIGINT/SIGTERM) or context cancellation,
// then performs graceful shutdown (HTTP close → jobs stop).
//
// Configuration is loaded from:
//   - Global flags (--workspace-dir, --config, etc.)
//   - Server-specific flags (--addr, --port, --no-ui, --no-api, --jobs-concurrency)
//   - Environment variables (PENTORA_*)
//   - Config file (pentora.yaml)
//
// Example usage:
//
//	pentora server start
//	pentora server start --addr 0.0.0.0 --port 8080
//	pentora server start --workspace-dir /data/pentora --jobs-concurrency 10
//
// See NOTES.md#30 for detailed server architecture design.
func newStartServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Pentora server",
		Long: `Start the Pentora server process.

The server hosts multiple components in a single runtime:
  - HTTP API (REST endpoints for scan management and workspace queries)
  - Web UI (static SPA with client-side routing)
  - Background workers (job queue, scheduler, notifier)

The server runs until interrupted (Ctrl+C) or killed, performing graceful
shutdown to drain in-flight requests and complete running jobs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Extract configuration from cmd.Context() (AppManager config)
			// TODO: Wire flags to config.Server struct (--addr, --port, etc.)
			// TODO: Initialize dependencies (workspace, engine manager, logging)
			// TODO: Pass config and deps to server.NewServer()

			// Create server instance with nil config (temporary - will be wired properly)
			server := server.NewServer(nil)

			// Start the server (begins HTTP listener and background workers)
			// Uses context from CLI root for cancellation/shutdown signaling
			server.Start(cmd.Context())

			// Block until server shutdown completes
			server.Wait()

			return nil
		},
	}

	// TODO: Add server-specific flags:
	// cmd.Flags().String("addr", "127.0.0.1", "Server listen address")
	// cmd.Flags().Int("port", 8080, "Server listen port")
	// cmd.Flags().Bool("no-ui", false, "Disable UI static serving")
	// cmd.Flags().Bool("no-api", false, "Disable REST API endpoints")
	// cmd.Flags().Int("jobs-concurrency", 4, "Number of concurrent background workers")
	// cmd.Flags().Duration("read-timeout", 30*time.Second, "HTTP read timeout")
	// cmd.Flags().Duration("write-timeout", 30*time.Second, "HTTP write timeout")

	return cmd
}
