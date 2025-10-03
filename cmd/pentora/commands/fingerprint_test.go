package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFingerprintSyncCommandFromFile(t *testing.T) {
	cmd := newFingerprintSyncCommand()
	tempDir := t.TempDir()
	catalogPath := filepath.Join("..", "..", "..", "pkg", "fingerprint", "data", "probes.yaml")

	cmd.SetArgs([]string{"--file", catalogPath, "--cache-dir", tempDir})
	cmd.SetContext(context.Background())

	if err := cmd.Execute(); err != nil {
		t.Fatalf("sync command failed: %v", err)
	}

	cached := filepath.Join(tempDir, "probe.catalog.yaml")
	if _, err := os.Stat(cached); err != nil {
		t.Fatalf("expected cached catalog at %s: %v", cached, err)
	}
}
