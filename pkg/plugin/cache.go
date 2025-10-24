// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// normalizePluginName converts a display name to a filesystem-safe slug.
// This ensures consistent directory names regardless of spaces or special characters.
//
// Examples:
//   - "SSH Weak Cipher" → "ssh-weak-cipher"
//   - "HTTP/2 Detection" → "http-2-detection"
//   - "Multiple   Spaces" → "multiple-spaces"
func normalizePluginName(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and slashes with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "-")

	// Remove multiple consecutive hyphens
	re := regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}

// CacheManager manages plugin download cache.
// Cache location: ~/.pentora/plugins/cache/
type CacheManager struct {
	// Base cache directory (e.g., ~/.pentora/plugins/cache/)
	cacheDir string

	// Registry for tracking cached plugins
	registry *YAMLRegistry
}

// NewCacheManager creates a new cache manager.
// It scans the cache directory and loads existing plugins into the registry.
func NewCacheManager(cacheDir string) (*CacheManager, error) {
	if cacheDir == "" {
		return nil, fmt.Errorf("cache directory cannot be empty")
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cm := &CacheManager{
		cacheDir: cacheDir,
		registry: NewYAMLRegistry(),
	}

	// Load existing plugins from disk into registry
	// This prevents re-downloading already cached plugins
	_, _ = cm.LoadFromDisk() // Ignore errors - partial load is acceptable

	return cm, nil
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
	// Use normalized name for filesystem (e.g., "SSH Weak Cipher" → "ssh-weak-cipher")
	normalizedName := normalizePluginName(plugin.Name)
	pluginDir := filepath.Join(c.cacheDir, normalizedName, plugin.Version)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create plugin cache directory: %w", err)
	}

	// Cache file path
	cachePath := filepath.Join(pluginDir, "plugin.yaml")

	// Write plugin to disk (so GetEntry can find it later)
	data, err := yaml.Marshal(plugin)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugin: %w", err)
	}
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write plugin to cache: %w", err)
	}

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

// GetEntry retrieves a cache entry by name and version.
func (c *CacheManager) GetEntry(name, version string) (*CacheEntry, error) {
	plugin, found := c.registry.Get(name)
	if !found {
		return nil, fmt.Errorf("plugin '%s' not found in cache", name)
	}

	if plugin.Version != version {
		return nil, fmt.Errorf("plugin '%s' found but version mismatch: expected %s, got %s", name, version, plugin.Version)
	}

	// Use normalized name for filesystem path lookup
	normalizedName := normalizePluginName(name)
	pluginDir := filepath.Join(c.cacheDir, normalizedName, version)
	cachePath := filepath.Join(pluginDir, "plugin.yaml")

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cache file not found for plugin '%s' version '%s'", name, version)
	}

	// Get file info for timestamps
	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat cache file: %w", err)
	}

	entry := &CacheEntry{
		Name:     name,
		Version:  version,
		Path:     cachePath,
		CachedAt: info.ModTime(),
		LastUsed: info.ModTime(),
	}

	return entry, nil
}

// Remove removes a plugin from the cache.
func (c *CacheManager) Remove(name string, version string) error {
	// Check if cache directory exists for this version
	// Use normalized name for filesystem path
	normalizedName := normalizePluginName(name)
	pluginDir := filepath.Join(c.cacheDir, normalizedName, version)
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin '%s' version '%s' not found in cache", name, version)
	}

	// Remove cache directory for this specific version
	if err := os.RemoveAll(pluginDir); err != nil {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	// Check if this is the version currently in the registry
	currentPlugin, found := c.registry.Get(name)
	if found && currentPlugin.Version == version {
		// Only unregister if we're removing the currently registered version
		if err := c.registry.Unregister(name); err != nil {
			return fmt.Errorf("failed to unregister plugin: %w", err)
		}
	}

	// Clean up parent directory if empty
	parentDir := filepath.Join(c.cacheDir, normalizedName)
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

// ListEntries returns all cache entries.
func (c *CacheManager) ListEntries() []*CacheEntry {
	plugins := c.registry.List()
	entries := make([]*CacheEntry, 0, len(plugins))

	for _, plugin := range plugins {
		entry, err := c.GetEntry(plugin.Name, plugin.Version)
		if err == nil {
			entries = append(entries, entry)
		}
	}

	return entries
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
