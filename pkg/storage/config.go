package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RetentionConfig defines scan retention policies for automatic cleanup.
//
// Both MaxScans and MaxAgeDays can be set simultaneously. When both are set,
// scans are deleted if they violate EITHER condition (OR logic).
//
// Examples:
//   - MaxScans=100, MaxAgeDays=0: Keep only the 100 most recent scans
//   - MaxScans=0, MaxAgeDays=30: Keep scans from last 30 days only
//   - MaxScans=100, MaxAgeDays=30: Keep max 100 scans AND delete scans older than 30 days
//
// A value of 0 means no limit for that policy.
type RetentionConfig struct {
	// MaxScans is the maximum number of scans to retain per organization.
	// Oldest scans are deleted first when this limit is exceeded.
	// 0 = no limit (default)
	MaxScans int `yaml:"max_scans" env:"PENTORA_RETENTION_MAX_SCANS"`

	// MaxAgeDays is the maximum age of scans in days.
	// Scans older than this are deleted.
	// 0 = no limit (default)
	MaxAgeDays int `yaml:"max_age_days" env:"PENTORA_RETENTION_MAX_AGE_DAYS"`
}

// IsEnabled returns true if any retention policy is configured.
func (r *RetentionConfig) IsEnabled() bool {
	return r.MaxScans > 0 || r.MaxAgeDays > 0
}

// Validate checks if the retention configuration is valid.
func (r *RetentionConfig) Validate() error {
	if r.MaxScans < 0 {
		return fmt.Errorf("max_scans must be >= 0, got %d", r.MaxScans)
	}
	if r.MaxAgeDays < 0 {
		return fmt.Errorf("max_age_days must be >= 0, got %d", r.MaxAgeDays)
	}
	return nil
}

// Config holds storage backend configuration.
//
// This configuration structure supports both OSS and Enterprise editions.
// OSS-specific fields are ignored by Enterprise, and vice versa.
type Config struct {
	// OSS Edition fields

	// WorkspaceRoot is the root directory for file-based storage (OSS).
	// Default: Platform-specific (see DefaultWorkspaceRoot())
	//   - Linux:   ~/.local/share/pentora
	//   - macOS:   ~/Library/Application Support/Pentora
	//   - Windows: %AppData%\Pentora
	WorkspaceRoot string `yaml:"workspace_root" env:"PENTORA_WORKSPACE"`

	// Retention policy configuration (applies to both OSS and Enterprise)
	Retention RetentionConfig `yaml:"retention"`

	// Enterprise Edition fields (ignored by OSS)

	// DatabaseURL is the PostgreSQL connection string (Enterprise).
	// Format: postgresql://user:password@host:port/database?sslmode=...
	DatabaseURL string `yaml:"database_url" env:"PENTORA_DATABASE_URL"`

	// S3Endpoint is the S3-compatible storage endpoint (Enterprise).
	// Examples: https://s3.amazonaws.com, http://minio:9000
	S3Endpoint string `yaml:"s3_endpoint" env:"PENTORA_S3_ENDPOINT"`

	// S3Region is the S3 region (Enterprise).
	S3Region string `yaml:"s3_region" env:"PENTORA_S3_REGION"`

	// S3Bucket is the S3 bucket name (Enterprise).
	S3Bucket string `yaml:"s3_bucket" env:"PENTORA_S3_BUCKET"`

	// S3AccessKey is the S3 access key ID (Enterprise).
	S3AccessKey string `yaml:"s3_access_key" env:"PENTORA_S3_ACCESS_KEY"`

	// S3SecretKey is the S3 secret access key (Enterprise).
	S3SecretKey string `yaml:"s3_secret_key" env:"PENTORA_S3_SECRET_KEY"`

	// S3UsePathStyle forces path-style S3 URLs (for Minio compatibility).
	S3UsePathStyle bool `yaml:"s3_use_path_style" env:"PENTORA_S3_USE_PATH_STYLE"`
}

// Validate checks if the configuration is valid for the current edition.
//
// OSS: Requires WorkspaceRoot
// Enterprise: Requires DatabaseURL, S3Bucket, S3Region
func (c *Config) Validate() error {
	// Validate retention policy
	if err := c.Retention.Validate(); err != nil {
		return fmt.Errorf("retention config: %w", err)
	}

	// This will be implemented differently based on edition
	// For now, just validate OSS requirements
	return c.validateOSS()
}

// validateOSS validates OSS-specific configuration.
func (c *Config) validateOSS() error {
	if c.WorkspaceRoot == "" {
		return NewInvalidInputError("workspace_root", "workspace root directory is required")
	}

	// Expand tilde in path
	if strings.HasPrefix(c.WorkspaceRoot, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		c.WorkspaceRoot = filepath.Join(home, c.WorkspaceRoot[2:])
	}

	// Make absolute path
	absPath, err := filepath.Abs(c.WorkspaceRoot)
	if err != nil {
		return NewInvalidInputError("workspace_root", fmt.Sprintf("invalid path: %v", err))
	}
	c.WorkspaceRoot = absPath

	return nil
}

// DefaultWorkspaceRoot returns the default workspace root for the current platform.
//
// Linux:   ~/.local/share/pentora
// macOS:   ~/Library/Application Support/Pentora
// Windows: %AppData%\Pentora
func DefaultWorkspaceRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch {
	case isWindows():
		// Windows: %AppData%\Pentora
		appData := os.Getenv("AppData")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Pentora"), nil

	case isDarwin():
		// macOS: ~/Library/Application Support/Pentora
		return filepath.Join(home, "Library", "Application Support", "Pentora"), nil

	default:
		// Linux/Unix: ~/.local/share/pentora
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" {
			xdgData = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(xdgData, "pentora"), nil
	}
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() (*Config, error) {
	workspaceRoot, err := DefaultWorkspaceRoot()
	if err != nil {
		return nil, err
	}

	return &Config{
		WorkspaceRoot: workspaceRoot,
	}, nil
}

// Platform detection helpers

// isWindows returns true if running on Windows.
func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

// isDarwin returns true if running on macOS.
func isDarwin() bool {
	// Check for macOS-specific environment variable
	// This is a heuristic; a more robust check would use runtime.GOOS
	return os.Getenv("HOME") != "" && fileExists("/System/Library/CoreServices/Finder.app")
}

// fileExists checks if a file or directory exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
