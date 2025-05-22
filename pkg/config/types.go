// pkg/config/types.go
package config

// Config is the root configuration structure for the Pentora application.
// It aggregates all other specific configuration structs.
type Config struct {
	Log LogConfig `description:"Logging configuration" koanf:"log"` // Logging configuration
}

// LogConfig holds logging related configuration.
type LogConfig struct {
	Level  string `description:"Log level set to pentora logs." koanf:"level"`   // Log level (e.g., "debug", "info", "warn", "error")
	Format string `description:"Pentora log format: json | text" koanf:"format"` // Log format (e.g., "json", "text")
	File   string `description:"Log file path" koanf:"file"`                     // Log file path (optional)
}
