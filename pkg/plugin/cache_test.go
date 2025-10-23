// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCacheManager(t *testing.T) {
	// Create temporary cache directory
	cacheDir := t.TempDir()

	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)
	require.NotNil(t, cm)
	require.Equal(t, cacheDir, cm.cacheDir)
	require.NotNil(t, cm.registry)

	// Verify directory was created
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestNewCacheManager_EmptyPath(t *testing.T) {
	cm, err := NewCacheManager("")
	require.Error(t, err)
	require.Nil(t, cm)
	require.Contains(t, err.Error(), "cache directory cannot be empty")
}

func TestCacheManager_Add(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"test"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	entry, err := cm.Add(plugin, "sha256:abc123", "https://example.com/plugin.yaml")
	require.NoError(t, err)
	require.NotNil(t, entry)
	require.Equal(t, "test-plugin", entry.Name)
	require.Equal(t, "1.0.0", entry.Version)
	require.Equal(t, "sha256:abc123", entry.Checksum)
	require.Equal(t, "https://example.com/plugin.yaml", entry.DownloadURL)
	require.False(t, entry.CachedAt.IsZero())
	require.False(t, entry.LastUsed.IsZero())

	// Verify directory was created
	expectedDir := filepath.Join(cacheDir, "test-plugin", "1.0.0")
	info, err := os.Stat(expectedDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())

	// Verify plugin in registry
	cachedPlugin, exists := cm.Get("test-plugin")
	require.True(t, exists)
	require.Equal(t, "test-plugin", cachedPlugin.Name)
}

func TestCacheManager_Add_NilPlugin(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	entry, err := cm.Add(nil, "sha256:abc", "https://example.com")
	require.Error(t, err)
	require.Nil(t, entry)
	require.Contains(t, err.Error(), "cannot cache nil plugin")
}

func TestCacheManager_Add_InvalidPlugin(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Plugin missing required fields
	plugin := &YAMLPlugin{
		Name: "invalid-plugin",
		// Missing version, type, author, etc.
	}

	entry, err := cm.Add(plugin, "sha256:abc", "https://example.com")
	require.Error(t, err)
	require.Nil(t, entry)
	require.Contains(t, err.Error(), "validation failed")
}

func TestCacheManager_Add_Duplicate(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"test"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	// Add first time
	_, err = cm.Add(plugin, "sha256:abc123", "https://example.com/v1.yaml")
	require.NoError(t, err)

	// Add again with different checksum (simulates update)
	plugin.Version = "1.0.1"
	entry, err := cm.Add(plugin, "sha256:def456", "https://example.com/v2.yaml")
	require.NoError(t, err)
	require.Equal(t, "sha256:def456", entry.Checksum)
}

func TestCacheManager_Get(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"test"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	_, err = cm.Add(plugin, "sha256:abc", "https://example.com")
	require.NoError(t, err)

	// Get existing plugin
	cached, exists := cm.Get("test-plugin")
	require.True(t, exists)
	require.Equal(t, "test-plugin", cached.Name)

	// Get non-existent plugin
	_, exists = cm.Get("non-existent")
	require.False(t, exists)
}

func TestCacheManager_Remove(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"test"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	_, err = cm.Add(plugin, "sha256:abc", "https://example.com")
	require.NoError(t, err)

	// Verify plugin exists
	_, exists := cm.Get("test-plugin")
	require.True(t, exists)

	// Remove plugin
	err = cm.Remove("test-plugin", "1.0.0")
	require.NoError(t, err)

	// Verify plugin is gone
	_, exists = cm.Get("test-plugin")
	require.False(t, exists)

	// Verify directory is removed
	pluginDir := filepath.Join(cacheDir, "test-plugin", "1.0.0")
	_, err = os.Stat(pluginDir)
	require.True(t, os.IsNotExist(err))
}

func TestCacheManager_Remove_NotFound(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	err = cm.Remove("non-existent", "1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestCacheManager_List(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Add multiple plugins
	for i := 1; i <= 3; i++ {
		plugin := &YAMLPlugin{
			Name:    "plugin-" + string(rune('0'+i)),
			Version: "1.0.0",
			Type:    EvaluationType,
			Author:  "test",
			Metadata: PluginMetadata{
				Severity: HighSeverity,
				Tags:     []string{"test"},
			},
			Output: OutputBlock{Message: "Test"},
		}
		_, err = cm.Add(plugin, "sha256:abc", "https://example.com")
		require.NoError(t, err)
	}

	plugins := cm.List()
	require.Len(t, plugins, 3)
}

func TestCacheManager_Clear(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Add plugins
	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"test"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	_, err = cm.Add(plugin, "sha256:abc", "https://example.com")
	require.NoError(t, err)

	// Verify plugin exists
	require.Len(t, cm.List(), 1)

	// Clear cache
	err = cm.Clear()
	require.NoError(t, err)

	// Verify cache is empty
	require.Len(t, cm.List(), 0)

	// Verify directories are removed
	entries, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestCacheManager_Size(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Initial size should be 0
	size, err := cm.Size()
	require.NoError(t, err)
	require.Equal(t, int64(0), size)

	// Create test file in cache
	testFile := filepath.Join(cacheDir, "test.txt")
	testData := []byte("Hello, World!")
	err = os.WriteFile(testFile, testData, 0o644)
	require.NoError(t, err)

	// Size should include test file
	size, err = cm.Size()
	require.NoError(t, err)
	require.Equal(t, int64(len(testData)), size)
}

func TestCacheManager_Prune(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Add old plugin
	oldPluginDir := filepath.Join(cacheDir, "old-plugin", "1.0.0")
	err = os.MkdirAll(oldPluginDir, 0o755)
	require.NoError(t, err)

	// Set modification time to 2 days ago
	oldTime := time.Now().Add(-48 * time.Hour)
	err = os.Chtimes(filepath.Join(cacheDir, "old-plugin"), oldTime, oldTime)
	require.NoError(t, err)

	// Add recent plugin
	recentPlugin := &YAMLPlugin{
		Name:    "recent-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"test"},
		},
		Output: OutputBlock{Message: "Test"},
	}
	_, err = cm.Add(recentPlugin, "sha256:abc", "https://example.com")
	require.NoError(t, err)

	// Prune plugins older than 24 hours
	removed, err := cm.Prune(24 * time.Hour)
	require.NoError(t, err)
	require.Equal(t, 1, removed)

	// Verify old plugin is removed
	_, err = os.Stat(oldPluginDir)
	require.True(t, os.IsNotExist(err))

	// Verify recent plugin still exists
	recentPluginDir := filepath.Join(cacheDir, "recent-plugin", "1.0.0")
	_, err = os.Stat(recentPluginDir)
	require.NoError(t, err)
}

func TestCacheManager_LoadFromDisk(t *testing.T) {
	cacheDir := t.TempDir()

	// Create test plugin files
	pluginDir := filepath.Join(cacheDir, "test-plugin", "1.0.0")
	err := os.MkdirAll(pluginDir, 0o755)
	require.NoError(t, err)

	pluginYAML := `name: test-plugin
version: 1.0.0
type: evaluation
author: test
metadata:
  severity: high
  tags:
    - test
output:
  vulnerability: true
  message: Test finding
`
	pluginFile := filepath.Join(pluginDir, "plugin.yaml")
	err = os.WriteFile(pluginFile, []byte(pluginYAML), 0o644)
	require.NoError(t, err)

	// Create cache manager
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Load from disk
	count, errors := cm.LoadFromDisk()
	require.Empty(t, errors)
	require.Equal(t, 1, count)

	// Verify plugin loaded
	plugin, exists := cm.Get("test-plugin")
	require.True(t, exists)
	require.Equal(t, "test-plugin", plugin.Name)
	require.Equal(t, "1.0.0", plugin.Version)
}

func TestCacheManager_LoadFromDisk_WithErrors(t *testing.T) {
	cacheDir := t.TempDir()

	// Create valid plugin
	validDir := filepath.Join(cacheDir, "valid-plugin", "1.0.0")
	err := os.MkdirAll(validDir, 0o755)
	require.NoError(t, err)

	validYAML := `name: valid-plugin
version: 1.0.0
type: evaluation
author: test
metadata:
  severity: high
  tags: [test]
output:
  message: Valid
`
	err = os.WriteFile(filepath.Join(validDir, "plugin.yaml"), []byte(validYAML), 0o644)
	require.NoError(t, err)

	// Create invalid plugin
	invalidDir := filepath.Join(cacheDir, "invalid-plugin", "1.0.0")
	err = os.MkdirAll(invalidDir, 0o755)
	require.NoError(t, err)

	invalidYAML := `name: invalid-plugin
version: 1.0.0
# Missing required fields
`
	err = os.WriteFile(filepath.Join(invalidDir, "plugin.yaml"), []byte(invalidYAML), 0o644)
	require.NoError(t, err)

	// Create cache manager
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Load from disk (partial success)
	count, errors := cm.LoadFromDisk()
	require.NotEmpty(t, errors)
	require.Equal(t, 1, count) // Only valid plugin loaded

	// Verify valid plugin loaded
	_, exists := cm.Get("valid-plugin")
	require.True(t, exists)
}

func TestNewCacheManager_CreateDirError(t *testing.T) {
	// Create a file where the cache directory should be
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file-not-dir")
	err := os.WriteFile(filePath, []byte("test"), 0o644)
	require.NoError(t, err)

	// Try to create cache manager with path that has a file in the way
	cachePath := filepath.Join(filePath, "cache")
	cm, err := NewCacheManager(cachePath)
	require.Error(t, err)
	require.Nil(t, cm)
	require.Contains(t, err.Error(), "failed to create cache directory")
}

func TestCacheManager_Add_MkdirError(t *testing.T) {
	// Create a file where plugin directory should be
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Create a file that will block mkdir
	blockingFile := filepath.Join(cacheDir, "blocked-plugin")
	err = os.WriteFile(blockingFile, []byte("blocking"), 0o644)
	require.NoError(t, err)

	plugin := &YAMLPlugin{
		Name:    "blocked-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"test"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	entry, err := cm.Add(plugin, "sha256:abc", "https://example.com")
	require.Error(t, err)
	require.Nil(t, entry)
	require.Contains(t, err.Error(), "failed to create plugin cache directory")
}

func TestCacheManager_Remove_NonExistentPlugin(t *testing.T) {
	cacheDir := t.TempDir()
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Try to remove plugin that doesn't exist in registry
	err = cm.Remove("nonexistent-plugin", "1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unregister plugin")
}

func TestCacheManager_Size_ReadDirError(t *testing.T) {
	// Create cache manager with non-existent directory
	cm := &CacheManager{
		cacheDir: "/nonexistent/path/that/does/not/exist",
		registry: NewYAMLRegistry(),
	}

	size, err := cm.Size()
	require.Error(t, err)
	require.Equal(t, int64(0), size)
}

func TestCacheManager_Clear_ReadDirError(t *testing.T) {
	// Create cache manager with non-existent directory
	cm := &CacheManager{
		cacheDir: "/nonexistent/path/that/does/not/exist",
		registry: NewYAMLRegistry(),
	}

	err := cm.Clear()
	require.Error(t, err)
}

func TestCacheManager_Prune_ReadDirError(t *testing.T) {
	// Create cache manager with non-existent directory
	cm := &CacheManager{
		cacheDir: "/nonexistent/path/that/does/not/exist",
		registry: NewYAMLRegistry(),
	}

	removed, err := cm.Prune(24 * time.Hour)
	require.Error(t, err)
	require.Equal(t, 0, removed)
}
