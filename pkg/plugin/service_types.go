// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import "time"

// Service layer types for plugin operations
// These types are used by the service layer to abstract business logic
// from the CMD and API layers.

// InstallOptions holds parameters for Install operation
type InstallOptions struct {
	// Source specifies which plugin source to use (empty = all sources)
	Source string

	// Force reinstalls plugins even if already cached
	Force bool

	// DryRun simulates installation without actually downloading
	DryRun bool

	// Category filter for bulk installs (optional)
	Category Category
}

// InstallResult holds results of Install operation
type InstallResult struct {
	// InstalledCount is the number of plugins successfully installed
	InstalledCount int

	// SkippedCount is the number of plugins already installed (not forced)
	SkippedCount int

	// FailedCount is the number of plugins that failed to install
	FailedCount int

	// Plugins contains information about installed plugins
	Plugins []*PluginInfo

	// Errors contains all errors encountered during installation
	// Collected for partial failure scenarios
	Errors []error
}

// UpdateOptions holds parameters for Update operation
type UpdateOptions struct {
	// Source specifies which plugin source to use (empty = all sources)
	Source string

	// Category filter for bulk updates (optional)
	Category Category

	// Force re-downloads plugins even if already cached
	Force bool

	// DryRun simulates update without actually downloading
	DryRun bool
}

// UpdateResult holds results of Update operation
type UpdateResult struct {
	// DownloadedCount is the number of plugins successfully downloaded
	DownloadedCount int

	// SkippedCount is the number of plugins already cached (not forced)
	SkippedCount int

	// FailedCount is the number of plugins that failed to download
	FailedCount int

	// TotalInCache is the total number of plugins in cache after update
	TotalInCache int

	// Errors contains all errors encountered during update
	Errors []error
}

// UninstallOptions holds parameters for Uninstall operation
type UninstallOptions struct {
	// All uninstalls all plugins if true
	All bool

	// Category filter for bulk uninstall (optional)
	Category Category
}

// UninstallResult holds results of Uninstall operation
type UninstallResult struct {
	// RemovedCount is the number of plugins successfully removed
	RemovedCount int

	// FailedCount is the number of plugins that failed to uninstall
	FailedCount int

	// RemainingCount is the number of plugins remaining after uninstall
	RemainingCount int

	// Errors contains all errors encountered during uninstall
	Errors []error
}

// PluginInfo holds detailed information about a plugin
// Used for List and GetInfo operations
type PluginInfo struct {
	// Plugin metadata
	ID       string
	Name     string
	Version  string
	Type     string
	Author   string
	Severity string
	Tags     []string

	// Installation info
	Checksum     string
	DownloadURL  string
	InstalledAt  time.Time
	LastVerified time.Time

	// File system info
	Path      string
	CacheDir  string
	CacheSize int64 // Size in bytes
}

// PluginSource represents a remote plugin repository
type PluginSource struct {
	// Name of the source (e.g., "official", "community")
	Name string `yaml:"name"`

	// URL of the manifest file
	URL string `yaml:"url"`

	// Enabled indicates if this source is active
	Enabled bool `yaml:"enabled"`

	// Priority for conflict resolution (lower number = higher priority)
	Priority int `yaml:"priority"`

	// Mirrors are alternative URLs for redundancy
	Mirrors []string `yaml:"mirrors,omitempty"`
}

// PluginManifestEntry describes a plugin in the remote manifest
type PluginManifestEntry struct {
	// Plugin metadata
	ID          string     `json:"id" yaml:"id"`         // Unique plugin identifier (slug)
	Name        string     `yaml:"name" json:"name"`
	Version     string     `yaml:"version" json:"version"`
	Description string     `yaml:"description" json:"description"`
	Author      string     `yaml:"author" json:"author"`

	// Categorization
	Categories []Category `yaml:"categories" json:"categories"`

	// Download info
	URL      string `yaml:"url" json:"url"`           // Download URL
	Checksum string `yaml:"checksum" json:"checksum"` // sha256:hex
	Size     int64  `yaml:"size" json:"size"`         // File size in bytes
}
