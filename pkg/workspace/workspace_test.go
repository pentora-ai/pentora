package workspace

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrepareCreatesStructure(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ws")

	prepared, err := Prepare(root)
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if prepared != root {
		t.Fatalf("expected %q, got %q", root, prepared)
	}

	for _, sub := range Subdirectories() {
		if info, err := os.Stat(filepath.Join(root, sub)); err != nil {
			t.Fatalf("subdir %q missing: %v", sub, err)
		} else if !info.IsDir() {
			t.Fatalf("subdir %q is not a directory", sub)
		}
	}
}

func TestPrepareUsesDefaultRoot(t *testing.T) {
	temp := t.TempDir()

	switch runtime.GOOS {
	case "windows":
		t.Setenv("AppData", temp)
	case "darwin":
		// macOS default uses home dir; override with env var for deterministic test.
		t.Setenv("PENTORA_WORKSPACE", filepath.Join(temp, "pentora"))
	default:
		t.Setenv("XDG_DATA_HOME", temp)
	}

	// Ensure explicit override is cleared when needed.
	if runtime.GOOS != "darwin" {
		t.Setenv("PENTORA_WORKSPACE", "")
	}

	prepared, err := Prepare("")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	if _, err := os.Stat(prepared); err != nil {
		t.Fatalf("default root not created: %v", err)
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	ctx = WithContext(ctx, "/tmp/ws")

	root, ok := FromContext(ctx)
	if !ok || root != "/tmp/ws" {
		t.Fatalf("expected workspace root /tmp/ws, got %q", root)
	}

	if _, ok := FromContext(context.Background()); ok {
		t.Fatalf("expected missing workspace root from empty context")
	}
}
