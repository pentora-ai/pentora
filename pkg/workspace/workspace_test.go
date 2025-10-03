package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/davecgh/go-spew/spew"
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

func TestPrepareInvalidRoot(t *testing.T) {
	old := userHomeDir
	defer func() { userHomeDir = old }()

	userHomeDir = func() (string, error) {
		return "", errors.New("cannot resolve home dir")
	}

	prepared, err := Prepare("")
	if err == nil {
		t.Fatalf("expected error, got prepared root %q", prepared)
	}
}

func TestPrepare_ErrCreateWorkspace(t *testing.T) {
	_, err := Prepare("/nonexistent/path")
	if err == nil {
		t.Fatalf("expected error, got prepared root %q", err)
	}
}

func TestPrepare_ErrCreateWorkspaceSubdir(t *testing.T) {
	tmp := t.TempDir()

	badSub := filepath.Join(tmp, defaultSubdirs[0])
	if err := os.WriteFile(badSub, []byte("not a dir"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Prepare(tmp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	t.Logf("got expected error: %v", err)
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
func TestWithContext_NilContext(t *testing.T) {
	//nolint:staticcheck
	ctx := WithContext(nil, "/tmp/ws")
	root, ok := FromContext(ctx)
	if !ok || root != "/tmp/ws" {
		t.Fatalf("expected workspace root /tmp/ws, got %q", root)
	}
}

func TestFromContext_NilContext(t *testing.T) {
	//nolint:staticcheck
	root, ok := FromContext(nil)
	if ok || root != "" {
		t.Fatalf("expected missing workspace root from nil context, got %q", root)
	}
}

func TestGetGOOS(t *testing.T) {
	expected := runtime.GOOS
	got := getGOOS()
	if got != expected {
		t.Fatalf("expected getGOOS to return '%s', got '%s'", expected, got)
	}
}

func TestDefaultRoot_DarwinError(t *testing.T) {
	old := userHomeDir
	defer func() { userHomeDir = old }()

	userHomeDir = func() (string, error) {
		return "", errors.New("cannot resolve home dir")
	}

	dir, err := defaultRoot()

	if err == nil && dir != "" {
		t.Fatalf("expected error, got %v", dir)
	}
}

func TestDefaultRoot_DarwinSuccess(t *testing.T) {
	oldGOOS := getGOOS
	defer func() { getGOOS = oldGOOS }()

	getGOOS = func() string { return "darwin" }

	oldUserHomeDir := userHomeDir
	defer func() { userHomeDir = oldUserHomeDir }()

	userHomeDir = func() (string, error) {
		return "/Users/testuser", nil
	}

	dir, err := defaultRoot()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := filepath.Join("/Users/testuser", "Library", "Application Support", "Pentora")
	if dir != expected {
		t.Fatalf("expected %s, got %s", expected, dir)
	}
}

func TestDefaultRoot_WindowsError(t *testing.T) {
	restoreGOOS := overrideGOOS(func() string {
		return "windows"
	})
	defer restoreGOOS()

	restoreHome := overrideUserHomeDir(func() (string, error) {
		return "C:\\Users\\testuser", nil
	})
	defer restoreHome()

	t.Setenv("AppData", "")

	dir, err := defaultRoot()

	spew.Dump(dir, err)

}
