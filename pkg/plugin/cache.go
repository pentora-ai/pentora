// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheManager manages plugin download cache.
// Cache location: ~/.pentora/plugins/cache/
type CacheManager struct {
	// Base cache directory (e.g., ~/.pentora/plugins/cache/)
	cacheDir string

	// Registry for tracking cached plugins
	registry *YAMLRegistry
}

// NewCacheManager creates a new cache manager.
func NewCacheManager(cacheDir string) (*CacheManager, error) {
	if cacheDir == "" {
		return nil, fmt.Errorf("cache directory cannot be empty")
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &CacheManager{
		cacheDir: cacheDir,
		registry: NewYAMLRegistry(),
	}, nil
}

// CacheEntry represents metadata about a cached plugin.
type CacheEntry struct {
	Name        string    // Plugin name
	Version     string    // Plugin version
	Path        string    // Path to cached YAML file
	Checksum    string    // SHA-256 checksum
	DownloadURL string    // Original download URL
	CachedAt    time.Time // When it was cached
	LastUsed    time.Time // Last access time
}

// Add adds a plugin to the cache.
// Returns the cache entry for the plugin.
func (c *CacheManager) Add(plugin *YAMLPlugin, checksum string, downloadURL string) (*CacheEntry, error) {
	if plugin == nil {
		return nil, fmt.Errorf("cannot cache nil plugin")
	}

	// Validate plugin before caching
	if err := plugin.Validate(); err != nil {
		return nil, fmt.Errorf("plugin validation failed: %w", err)
	}

	// Create plugin-specific cache directory
	// Structure: cache/<plugin-name>/<version>/plugin.yaml
	pluginDir := filepath.Join(c.cacheDir, plugin.Name, plugin.Version)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create plugin cache directory: %w", err)
	}

	// Cache file path
	cachePath := filepath.Join(pluginDir, "plugin.yaml")

	// Register in cache registry
	if err := c.registry.Register(plugin); err != nil {
		// If already exists, unregister and re-register
		_ = c.registry.Unregister(plugin.Name)
		if err := c.registry.Register(plugin); err != nil {
			return nil, fmt.Errorf("failed to register plugin in cache: %w", err)
		}
	}

	now := time.Now()
	entry := &CacheEntry{
		Name:        plugin.Name,
		Version:     plugin.Version,
		Path:        cachePath,
		Checksum:    checksum,
		DownloadURL: downloadURL,
		CachedAt:    now,
		LastUsed:    now,
	}

	return entry, nil
}

// Get retrieves a cached plugin by name.
func (c *CacheManager) Get(name string) (*YAMLPlugin, bool) {
	return c.registry.Get(name)
}

// Remove removes a plugin from the cache.
func (c *CacheManager) Remove(name string, version string) error {
	// Remove from registry
	if err := c.registry.Unregister(name); err != nil {
		return fmt.Errorf("failed to unregister plugin: %w", err)
	}

	// Remove cache directory
	pluginDir := filepath.Join(c.cacheDir, name, version)
	if err := os.RemoveAll(pluginDir); err != nil {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	// Clean up parent directory if empty
	parentDir := filepath.Join(c.cacheDir, name)
	entries, err := os.ReadDir(parentDir)
	if err == nil && len(entries) == 0 {
		_ = os.Remove(parentDir)
	}

	return nil
}

// List returns all cached plugins.
func (c *CacheManager) List() []*YAMLPlugin {
	return c.registry.List()
}

// Clear removes all cached plugins.
func (c *CacheManager) Clear() error {
	// Remove all plugin directories
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			pluginDir := filepath.Join(c.cacheDir, entry.Name())
			if err := os.RemoveAll(pluginDir); err != nil {
				return fmt.Errorf("failed to remove plugin directory %s: %w", entry.Name(), err)
			}
		}
	}

	// Clear registry
	c.registry.Clear()

	return nil
}

// Size returns the total size of the cache in bytes.
func (c *CacheManager) Size() (int64, error) {
	var totalSize int64

	err := filepath.Walk(c.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to calculate cache size: %w", err)
	}

	return totalSize, nil
}

// Prune removes unused cached plugins older than the specified duration.
// This is useful for cleaning up stale cache entries.
func (c *CacheManager) Prune(olderThan time.Duration) (int, error) {
	cutoffTime := time.Now().Add(-olderThan)
	removed := 0

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(c.cacheDir, entry.Name())

		// Check modification time
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			// Remove plugin from cache
			versions, err := os.ReadDir(pluginDir)
			if err != nil {
				continue
			}

			for _, versionEntry := range versions {
				if versionEntry.IsDir() {
					versionDir := filepath.Join(pluginDir, versionEntry.Name())
					if err := os.RemoveAll(versionDir); err != nil {
						continue
					}
					removed++
				}
			}

			// Remove plugin directory if empty
			remaining, err := os.ReadDir(pluginDir)
			if err == nil && len(remaining) == 0 {
				_ = os.Remove(pluginDir)
			}

			// Unregister from registry
			_ = c.registry.Unregister(entry.Name())
		}
	}

	return removed, nil
}

// LoadFromDisk loads all cached plugins from disk into the registry.
func (c *CacheManager) LoadFromDisk() (int, []error) {
	loader := NewLoader(c.cacheDir)
	plugins, err := loader.LoadRecursive(c.cacheDir)
	if err != nil {
		// Partial success - some plugins loaded, some failed
		loadedCount, regErrors := c.registry.RegisterBulk(plugins)
		var allErrors []error
		allErrors = append(allErrors, fmt.Errorf("load errors: %w", err))
		allErrors = append(allErrors, regErrors...)
		return loadedCount, allErrors
	}

	// All plugins loaded successfully
	return c.registry.RegisterBulk(plugins)
}
