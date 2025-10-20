package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var defaultSubdirs = []string{
	"scans",
	"queue",
	"cache",
	"logs",
	"reports",
	"audit",
}

var userHomeDir = os.UserHomeDir // for testing

// GetUserHomeDir returns the current user's home directory.
func GetUserHomeDir() (string, error) {
	return userHomeDir()
}

// Prepare ensures the workspace root and required subdirectories exist.
// It returns the absolute path to the workspace root that was prepared.
func Prepare(root string) (string, error) {
	if root == "" {
		var err error
		root, err = defaultRoot()
		if err != nil {
			return "", err
		}
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace path: %w", err)
	}

	if err := os.MkdirAll(absRoot, 0o750); err != nil {
		return "", fmt.Errorf("create workspace root: %w", err)
	}

	for _, sub := range defaultSubdirs {
		subPath := filepath.Join(absRoot, sub)
		if err := os.MkdirAll(subPath, 0o750); err != nil {
			return "", fmt.Errorf("create workspace subdir %q: %w", sub, err)
		}
	}

	return absRoot, nil
}

type ctxKey string

const workspaceRootKey ctxKey = "workspace.root"

// WithContext stores the prepared workspace root on the provided context.
func WithContext(ctx context.Context, root string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, workspaceRootKey, root)
}

// FromContext extracts the workspace root from context.
func FromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	val := ctx.Value(workspaceRootKey)
	if root, ok := val.(string); ok && root != "" {
		return root, true
	}
	return "", false
}

// getGOOS returns the current operating system target as a string,
// as defined by the Go runtime (e.g., "windows", "darwin", "linux").
// This function can be overridden for testing purposes.
var getGOOS = func() string {
	return runtime.GOOS
}

// defaultRoot determines the default root directory for the Pentora workspace.
// It checks for the "PENTORA_WORKSPACE" environment variable first. If not set,
// it selects a platform-specific default location:
//   - On macOS ("darwin"), it uses "$HOME/Library/Application Support/Pentora".
//   - On Windows, it prefers "%AppData%\Pentora", falling back to
//     "$HOME/AppData/Roaming/Pentora" if "AppData" is not set.
//   - On other systems (e.g., Linux), it uses "$XDG_DATA_HOME/pentora" if set,
//     otherwise defaults to "$HOME/.local/share/pentora".
//
// Returns the resolved directory path or an error if the home directory cannot be determined.
func defaultRoot() (string, error) {
	if dir := os.Getenv("PENTORA_WORKSPACE"); dir != "" {
		return dir, nil
	}

	switch getGOOS() {
	case "darwin":
		home, err := GetUserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "Pentora"), nil
	case "windows":
		if appData := os.Getenv("AppData"); appData != "" {
			return filepath.Join(appData, "Pentora"), nil
		}
		home, err := GetUserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, "AppData", "Roaming", "Pentora"), nil
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "pentora"), nil
		}
		home, err := GetUserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if home == "" {
			return "", errors.New("cannot determine workspace directory")
		}
		return filepath.Join(home, ".local", "share", "pentora"), nil
	}
}

// Subdirectories returns the list of default workspace subdirectories.
func Subdirectories() []string {
	subs := make([]string, len(defaultSubdirs))
	copy(subs, defaultSubdirs)
	return subs
}
