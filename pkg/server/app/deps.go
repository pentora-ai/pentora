package app

import (
	"github.com/rs/zerolog"

	"github.com/vulntor/vulntor/pkg/config"
	"github.com/vulntor/vulntor/pkg/server/api"
	"github.com/vulntor/vulntor/pkg/storage"
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

	// Logger for structured logging (injected by caller)
	Logger zerolog.Logger
}
