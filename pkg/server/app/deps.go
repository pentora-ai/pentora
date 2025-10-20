package app

import (
	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/server/api"
	"github.com/rs/zerolog"
)

// Deps holds dependencies for the server application.
// This pattern enables dependency injection and easier testing.
type Deps struct {
	// Workspace provides access to scan data
	Workspace api.WorkspaceInterface

	// Config manager for runtime configuration
	Config *config.Manager

	// Logger for structured logging
	Logger *zerolog.Logger
}
