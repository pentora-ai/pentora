package app

import (
	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/server/api"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/rs/zerolog"
)

// Deps holds dependencies for the server application.
// This pattern enables dependency injection and easier testing.
type Deps struct {
	// Storage backend for scan data (replaces Workspace)
	// Uses file-based storage (OSS) or PostgreSQL+S3 (Enterprise)
	Storage storage.Backend

	// Workspace provides access to scan data (DEPRECATED: use Storage instead)
	// Kept for backward compatibility during migration
	Workspace api.WorkspaceInterface

	// PluginService provides plugin management operations
	// Actual type: *plugin.Service (must implement v1.PluginService interface)
	// Type asserted in router to v1.PluginService
	PluginService any

	// Config manager for runtime configuration
	Config *config.Manager

	// Logger for structured logging
	Logger *zerolog.Logger
}
