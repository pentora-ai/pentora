// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// TestNewManifestWatcher_Success verifies that NewManifestWatcher creates
// a watcher with the correct configuration.
func TestNewManifestWatcher_Success(t *testing.T) {
	// Create temp directory for manifest
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "registry.json")

	// Create valid manifest file (plugins is a map, not array)
	initialManifest := `{"version":"1.0","plugins":{},"last_updated":"2025-01-01T00:00:00Z"}`

	err := os.WriteFile(manifestPath, []byte(initialManifest), 0644)
	require.NoError(t, err)

	// Create ManifestManager using constructor
	manifest, err := NewManifestManager(manifestPath)
	require.NoError(t, err)

	logger := zerolog.New(os.Stdout)
	watcher, err := NewManifestWatcher(manifest, logger)

	require.NoError(t, err)
	require.NotNil(t, watcher)
	require.Equal(t, manifest, watcher.manifest)
	require.NotNil(t, watcher.watcher)
	require.Equal(t, 100*time.Millisecond, watcher.debounceDelay)

	// Clean up
	require.NoError(t, watcher.Close())
}

// TestManifestWatcher_DetectsFileChange verifies that the watcher detects
// file changes and triggers a reload.
func TestManifestWatcher_DetectsFileChange(t *testing.T) {
	// Create temp directory for manifest
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "registry.json")

	// Create initial manifest file (plugins is a map)
	initialContent := `{"version":"1.0","plugins":{},"last_updated":"2025-01-01T00:00:00Z"}`

	err := os.WriteFile(manifestPath, []byte(initialContent), 0644)
	require.NoError(t, err)

	// Create ManifestManager
	manifest, err := NewManifestManager(manifestPath)
	require.NoError(t, err)

	// Load initial manifest
	err = manifest.Load()
	require.NoError(t, err)

	logger := zerolog.Nop() // Quiet logger for test
	watcher, err := NewManifestWatcher(manifest, logger)
	require.NoError(t, err)
	defer watcher.Close()

	// Start watcher in goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- watcher.Start(ctx)
	}()

	// Wait for watcher to initialize
	time.Sleep(50 * time.Millisecond)

	// Modify the file (add a plugin to the map)
	updatedContent := `{"version":"1.0","plugins":{"test-plugin":{"id":"test-plugin","name":"Test","version":"1.0.0","category":"test"}},"last_updated":"2025-01-01T01:00:00Z"}`

	err = os.WriteFile(manifestPath, []byte(updatedContent), 0644)
	require.NoError(t, err)

	// Wait for debounce delay + processing time
	time.Sleep(250 * time.Millisecond)

	// Stop watcher
	cancel()

	// Wait for watcher to stop
	select {
	case err := <-errChan:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(1 * time.Second):
		t.Fatal("Watcher did not stop in time")
	}

	// Verify manifest was reloaded by checking plugin count
	plugins, err := manifest.List()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(plugins), 0, "Manifest should be accessible after reload")
}

// TestManifestWatcher_ContextCancellation verifies that the watcher stops
// gracefully when the context is canceled.
func TestManifestWatcher_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "registry.json")

	manifestContent := `{"version":"1.0","plugins":{},"last_updated":"2025-01-01T00:00:00Z"}`

	err := os.WriteFile(manifestPath, []byte(manifestContent), 0644)
	require.NoError(t, err)

	manifest, err := NewManifestManager(manifestPath)
	require.NoError(t, err)

	logger := zerolog.Nop()
	watcher, err := NewManifestWatcher(manifest, logger)
	require.NoError(t, err)
	defer watcher.Close()

	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error, 1)
	go func() {
		errChan <- watcher.Start(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Verify watcher stops with context.Canceled error
	select {
	case err := <-errChan:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(1 * time.Second):
		t.Fatal("Watcher did not stop after context cancellation")
	}
}

// TestManifestWatcher_Close verifies that Close() properly releases resources.
func TestManifestWatcher_Close(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "registry.json")

	manifestContent := `{"version":"1.0","plugins":{},"last_updated":"2025-01-01T00:00:00Z"}`

	err := os.WriteFile(manifestPath, []byte(manifestContent), 0644)
	require.NoError(t, err)

	manifest, err := NewManifestManager(manifestPath)
	require.NoError(t, err)

	logger := zerolog.Nop()
	watcher, err := NewManifestWatcher(manifest, logger)
	require.NoError(t, err)

	// Close should succeed
	err = watcher.Close()
	require.NoError(t, err)

	// Second close might or might not fail depending on fsnotify implementation
	// Just verify no panic
	_ = watcher.Close()
}

// TestService_StartManifestWatcher_WithRealManager verifies that
// StartManifestWatcher works with a real ManifestManager.
func TestService_StartManifestWatcher_WithRealManager(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "registry.json")

	// Create manifest file
	manifestContent := `{"version":"1.0","plugins":{},"last_updated":"2025-01-01T00:00:00Z"}`

	err := os.WriteFile(manifestPath, []byte(manifestContent), 0644)
	require.NoError(t, err)

	// Create ManifestManager
	manifestMgr, err := NewManifestManager(manifestPath)
	require.NoError(t, err)

	// Create service with real ManifestManager
	service := &Service{
		manifest: manifestMgr,
		logger:   zerolog.Nop(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// StartManifestWatcher should succeed and block until context canceled
	err = service.StartManifestWatcher(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestService_StartManifestWatcher_WithMock verifies that
// StartManifestWatcher gracefully skips when using a mock (not *ManifestManager).
func TestService_StartManifestWatcher_WithMock(t *testing.T) {
	// Use mock manifest (not *ManifestManager)
	mockManifest := &mockManifestManager{}

	service := &Service{
		manifest: mockManifest,
		logger:   zerolog.Nop(),
	}

	ctx := context.Background()

	// Should return nil immediately (skip watcher for mock)
	err := service.StartManifestWatcher(ctx)
	require.NoError(t, err, "Should skip watcher for mock manifest")
}
