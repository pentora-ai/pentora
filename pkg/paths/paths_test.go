package paths

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestConfigDir(t *testing.T) {
	t.Run("XDGOverride", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-config")
		got := ConfigDir()
		want := filepath.Join("/tmp/xdg-config", "vulntor")
		if got != want {
			t.Fatalf("ConfigDir() = %s, want %s", got, want)
		}
	})

	t.Run("PlatformDefault", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		switch runtime.GOOS {
		case "windows":
			t.Setenv("AppData", `C:\AppData`)
			want := filepath.Join(`C:\AppData`, "Vulntor")
			if got := ConfigDir(); got != want {
				t.Fatalf("ConfigDir() = %s, want %s", got, want)
			}
		default:
			t.Setenv("HOME", "/home/tester")
			want := filepath.Join("/home/tester", ".config", "vulntor")
			if got := ConfigDir(); got != want {
				t.Fatalf("ConfigDir() = %s, want %s", got, want)
			}
		}
	})
}

func TestDataDir(t *testing.T) {
	t.Run("XDGOverride", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")
		got := DataDir()
		want := filepath.Join("/tmp/xdg-data", "vulntor")
		if got != want {
			t.Fatalf("DataDir() = %s, want %s", got, want)
		}
	})

	t.Run("PlatformDefault", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		switch runtime.GOOS {
		case "windows":
			t.Setenv("AppData", `C:\AppData`)
			want := filepath.Join(`C:\AppData`, "Vulntor")
			if got := DataDir(); got != want {
				t.Fatalf("DataDir() = %s, want %s", got, want)
			}
		default:
			t.Setenv("HOME", "/home/tester")
			want := filepath.Join("/home/tester", ".local", "share", "vulntor")
			if got := DataDir(); got != want {
				t.Fatalf("DataDir() = %s, want %s", got, want)
			}
		}
	})
}

func TestCacheDir(t *testing.T) {
	t.Run("XDGOverride", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "/tmp/xdg-cache")
		got := CacheDir()
		want := filepath.Join("/tmp/xdg-cache", "vulntor")
		if got != want {
			t.Fatalf("CacheDir() = %s, want %s", got, want)
		}
	})

	t.Run("PlatformDefault", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		switch runtime.GOOS {
		case "windows":
			t.Setenv("LocalAppData", `C:\LocalAppData`)
			want := filepath.Join(`C:\LocalAppData`, "Vulntor", "Cache")
			if got := CacheDir(); got != want {
				t.Fatalf("CacheDir() = %s, want %s", got, want)
			}
		default:
			t.Setenv("HOME", "/home/tester")
			want := filepath.Join("/home/tester", ".cache", "vulntor")
			if got := CacheDir(); got != want {
				t.Fatalf("CacheDir() = %s, want %s", got, want)
			}
		}
	})
}
