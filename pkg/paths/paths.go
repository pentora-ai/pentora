package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the config directory for Vulntor.
// Order: XDG_CONFIG_HOME/vulntor, platform-specific fallback.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "vulntor")
	}
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("AppData"); appData != "" {
			return filepath.Join(appData, "Vulntor")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "vulntor")
}

// DataDir returns the data directory for Vulntor.
// Order: XDG_DATA_HOME/vulntor, platform-specific fallback.
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "vulntor")
	}
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("AppData"); appData != "" {
			return filepath.Join(appData, "Vulntor")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "vulntor")
}

// CacheDir returns the cache directory for Vulntor.
// Order: XDG_CACHE_HOME/vulntor, platform-specific fallback.
func CacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "vulntor")
	}
	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LocalAppData"); localAppData != "" {
			return filepath.Join(localAppData, "Vulntor", "Cache")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "vulntor")
}
