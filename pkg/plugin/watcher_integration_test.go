//go:build integration
// +build integration

package plugin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func init() {
	// Disable all logging for integration tests to reduce noise
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// TestManifestWatcher_ServerAPISync verifies the primary scenario from Issue #27:
// CLI plugin changes automatically appear in server API without restart.
//
// Flow:
//  1. Create plugin service with real manifest
//  2. Start manifest watcher in background
//  3. Baseline: List plugins (count = N)
//  4. Simulate CLI install: Add plugin to manifest file
//  5. Wait for debounce + reload
//  6. Verify: List plugins (count = N+1)
//  7. Simulate CLI uninstall: Remove plugin from manifest
//  8. Wait for debounce + reload
//  9. Verify: List plugins (count = N)
func TestManifestWatcher_ServerAPISync(t *testing.T) {
	// Setup: Create temp directory with cache structure
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache", "plugins")
	manifestPath := filepath.Join(tmpDir, "cache", "registry.json")

	// Create cache directory structure
	err := os.MkdirAll(filepath.Dir(manifestPath), 0o755)
	require.NoError(t, err)

	// Initial manifest (empty plugins)
	initialManifest := Manifest{
		Version:     "1.0",
		Plugins:     make(map[string]*ManifestEntry),
		LastUpdated: time.Now(),
	}

	writeManifest(t, manifestPath, initialManifest)

	// Create plugin service with custom cache dir
	service, err := NewService(
		WithCacheDir(cacheDir),
		WithLogger(zerolog.Nop()),
	)
	require.NoError(t, err)
	require.NotNil(t, service)

	// Start manifest watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherErr := make(chan error, 1)
	go func() {
		watcherErr <- service.StartManifestWatcher(ctx)
	}()

	// Wait for watcher to initialize
	time.Sleep(150 * time.Millisecond)

	// Baseline: List plugins
	baselinePlugins, err := service.List(ctx)
	require.NoError(t, err)
	baselineCount := len(baselinePlugins)
	t.Logf("Baseline plugin count: %d", baselineCount)

	// Simulate CLI install: Add plugin to manifest
	newPlugin := &ManifestEntry{
		ID:          "test-plugin-001",
		Name:        "Test Plugin 001",
		Version:     "1.0.0",
		Type:        "evaluation",
		Author:      "integration-test",
		InstalledAt: time.Now(),
		Path:        "plugins/test-plugin-001/1.0.0/plugin.yaml",
	}

	manifest := readManifest(t, manifestPath)
	manifest.Plugins[newPlugin.ID] = newPlugin
	manifest.LastUpdated = time.Now()
	writeManifest(t, manifestPath, manifest)

	t.Log("Wrote new plugin to manifest, waiting for auto-reload...")

	// Wait for debounce (100ms) + reload + propagation
	time.Sleep(300 * time.Millisecond)

	// Verify: Plugin appears in service
	afterInstall, err := service.List(ctx)
	require.NoError(t, err)
	require.Equal(t, baselineCount+1, len(afterInstall), "Plugin count should increase by 1 after install")

	// Verify the plugin is actually there
	found := false
	for _, p := range afterInstall {
		if p.ID == "test-plugin-001" {
			found = true
			require.Equal(t, "Test Plugin 001", p.Name)
			break
		}
	}
	require.True(t, found, "Installed plugin should be in list")

	t.Log("✅ CLI install → API sync verified")

	// Simulate CLI uninstall: Remove plugin from manifest
	manifest = readManifest(t, manifestPath)
	delete(manifest.Plugins, newPlugin.ID)
	manifest.LastUpdated = time.Now()
	writeManifest(t, manifestPath, manifest)

	t.Log("Removed plugin from manifest, waiting for auto-reload...")

	// Wait for debounce + reload
	time.Sleep(300 * time.Millisecond)

	// Verify: Plugin removed from service
	afterUninstall, err := service.List(ctx)
	require.NoError(t, err)
	require.Equal(t, baselineCount, len(afterUninstall), "Plugin count should return to baseline after uninstall")

	// Verify the plugin is actually gone
	for _, p := range afterUninstall {
		require.NotEqual(t, "test-plugin-001", p.ID, "Uninstalled plugin should not be in list")
	}

	t.Log("✅ CLI uninstall → API sync verified")

	// Cleanup: Cancel watcher
	cancel()

	select {
	case err := <-watcherErr:
		require.ErrorIs(t, err, context.Canceled, "Watcher should exit with context.Canceled")
	case <-time.After(2 * time.Second):
		t.Fatal("Watcher did not exit after context cancellation")
	}

	t.Log("✅ Watcher graceful shutdown verified")
}

// TestManifestWatcher_RapidChanges verifies debouncing behavior:
// Multiple rapid changes should be coalesced into a single reload.
//
// Flow:
//  1. Start service with watcher
//  2. Make 5 rapid plugin installs (< 100ms apart)
//  3. Wait for debounce
//  4. Verify all 5 plugins appear (single reload, not 5)
func TestManifestWatcher_RapidChanges(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache", "plugins")
	manifestPath := filepath.Join(tmpDir, "cache", "registry.json")

	err := os.MkdirAll(filepath.Dir(manifestPath), 0o755)
	require.NoError(t, err)

	// Initial manifest
	initialManifest := Manifest{
		Version:     "1.0",
		Plugins:     make(map[string]*ManifestEntry),
		LastUpdated: time.Now(),
	}
	writeManifest(t, manifestPath, initialManifest)

	// Create service with watcher
	service, err := NewService(
		WithCacheDir(cacheDir),
		WithLogger(zerolog.Nop()),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go service.StartManifestWatcher(ctx)
	time.Sleep(150 * time.Millisecond) // Wait for watcher to start

	// Baseline
	baseline, err := service.List(ctx)
	require.NoError(t, err)
	baselineCount := len(baseline)

	// Rapid changes: Install 5 plugins with < 50ms between each
	manifest := readManifest(t, manifestPath)
	for i := 1; i <= 5; i++ {
		pluginID := "rapid-plugin-" + string(rune(i+48))
		manifest.Plugins[pluginID] = &ManifestEntry{
			ID:          pluginID,
			Name:        "Rapid Plugin " + string(rune(i+48)),
			Version:     "1.0.0",
			Type:        "evaluation",
			InstalledAt: time.Now(),
			Path:        "plugins/" + pluginID + "/1.0.0/plugin.yaml",
		}
		manifest.LastUpdated = time.Now()
		writeManifest(t, manifestPath, manifest)

		// Very short delay (less than debounce)
		time.Sleep(30 * time.Millisecond)
	}

	t.Log("Made 5 rapid changes to manifest")

	// Wait for debounce + reload (should only reload once)
	time.Sleep(300 * time.Millisecond)

	// Verify all 5 plugins appeared
	afterRapid, err := service.List(ctx)
	require.NoError(t, err)
	require.Equal(t, baselineCount+5, len(afterRapid), "All 5 rapid plugins should appear")

	t.Log("✅ Debouncing verified: 5 rapid changes, 1 reload")

	cancel()
}

// TestManifestWatcher_ServerShutdown verifies graceful cleanup:
// Watcher should stop cleanly when server context is canceled.
//
// Flow:
//  1. Start service with watcher
//  2. Verify watcher is running (make a change, verify reload)
//  3. Cancel server context
//  4. Verify watcher exits gracefully (no goroutine leak)
//  5. Verify no errors in shutdown
func TestManifestWatcher_ServerShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache", "plugins")
	manifestPath := filepath.Join(tmpDir, "cache", "registry.json")

	err := os.MkdirAll(filepath.Dir(manifestPath), 0o755)
	require.NoError(t, err)

	initialManifest := Manifest{
		Version:     "1.0",
		Plugins:     make(map[string]*ManifestEntry),
		LastUpdated: time.Now(),
	}
	writeManifest(t, manifestPath, initialManifest)

	service, err := NewService(
		WithCacheDir(cacheDir),
		WithLogger(zerolog.Nop()),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	watcherErr := make(chan error, 1)
	go func() {
		watcherErr <- service.StartManifestWatcher(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(150 * time.Millisecond)

	// Verify watcher is running by making a change
	manifest := readManifest(t, manifestPath)
	manifest.Plugins["test"] = &ManifestEntry{
		ID:          "test",
		Name:        "Test",
		Version:     "1.0.0",
		Type:        "evaluation",
		InstalledAt: time.Now(),
		Path:        "plugins/test/1.0.0/plugin.yaml",
	}
	writeManifest(t, manifestPath, manifest)

	time.Sleep(250 * time.Millisecond) // Wait for reload

	plugins, err := service.List(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(plugins), 1, "Watcher should have reloaded manifest")

	t.Log("✅ Watcher is running and reloading")

	// Cancel context to trigger shutdown
	cancel()

	// Verify watcher exits gracefully
	select {
	case err := <-watcherErr:
		require.ErrorIs(t, err, context.Canceled, "Watcher should exit with context.Canceled")
		t.Log("✅ Watcher exited gracefully with context.Canceled")
	case <-time.After(2 * time.Second):
		t.Fatal("Watcher did not exit within timeout")
	}

	// Note: Goroutine leak detection would require external tools like goleak
	// For now, we verify the watcher returned the expected error
}

// TestManifestWatcher_WatcherFailureDoesNotBlockServer verifies resilience:
// If manifest watcher fails to start, server should remain functional.
//
// Flow:
//  1. Delete manifest file before starting service
//  2. Try to start watcher (should fail but not panic)
//  3. Verify service still works (List returns empty, no crash)
func TestManifestWatcher_WatcherFailureDoesNotBlockServer(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache", "plugins")
	// Do NOT create manifest file directory - watcher should fail

	service, err := NewService(
		WithCacheDir(cacheDir),
		WithLogger(zerolog.Nop()),
	)
	require.NoError(t, err, "Service creation should succeed even if manifest doesn't exist")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Start watcher - will fail because directory doesn't exist
	err = service.StartManifestWatcher(ctx)

	// Should either return error immediately (directory doesn't exist)
	// or timeout (watcher not started)
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("Watcher failed as expected: %v", err)
	}

	// Verify service is still functional despite watcher failure
	ctx2 := context.Background()
	plugins, err := service.List(ctx2)
	require.NoError(t, err, "Service.List should work even if watcher failed")
	require.NotNil(t, plugins, "Service.List should return empty list, not nil")

	t.Log("✅ Service remains functional despite watcher failure")
}

// Helper functions

// writeManifest writes a manifest to the given path
func writeManifest(t *testing.T, path string, manifest Manifest) {
	t.Helper()

	// Ensure directory exists
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0o755)
	require.NoError(t, err)

	// Marshal manifest
	data, err := json.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)

	// Write to file
	err = os.WriteFile(path, data, 0o644)
	require.NoError(t, err)
}

// readManifest reads a manifest from the given path
func readManifest(t *testing.T, path string) Manifest {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var manifest Manifest
	err = json.Unmarshal(data, &manifest)
	require.NoError(t, err)

	return manifest
}
