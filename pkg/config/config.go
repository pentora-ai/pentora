// pkg/config/config.go
package config

import (
	"fmt"
	"sync"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

// Global Koanf instance, initialized once at startup.
var (
	k    *koanf.Koanf
	once sync.Once
)

// InitGlobalKoanf initializes the global Koanf instance.
// This should be called early in the application lifecycle, before Load.
func InitGlobalConfig() {
	once.Do(func() {
		k = koanf.New(".")
	})
}

// ConfigManager handles loading and accessing application configuration.
type Manager struct {
	koanfInstance *koanf.Koanf
	currentConfig Config
	mu            sync.RWMutex // To protect currentConfig during runtime updates
}

// NewManager creates a new ConfigManager.
// It initializes the global Koanf instance if not already done.
func NewManager( /*dbProvider dbprovider.Provider*/ ) *Manager { // Pass DB provider if used
	InitGlobalConfig() // Ensure global k is initialized
	// Initialize the Koanf instance if it hasn't been done already
	return &Manager{
		koanfInstance: k, // Use the global instance
		// dbProvider:    dbProvider,
	}
}

// DefaultConfig returns a new Config struct populated with hardcoded default values.
// These serve as the baseline configuration if no other sources override them.
func DefaultConfig() Config {
	return Config{
		Log: LogConfig{
			Level:  "info", // Default log level
			Format: "text", // Default log format
			File:   "",     // Default log file path
		},
		Server: DefaultServerConfig(),
	}
}

// Load loads configuration from various sources based on precedence.
// It populates the manager's currentConfig.
func (m *Manager) Load(flags *pflag.FlagSet, customConfigFilePath string) error {
	m.mu.Lock() // Lock for writing to m.koanfInstance and m.currentConfig
	defer m.mu.Unlock()

	defaultCfgMap := DefaultConfigAsMap()
	if err := m.koanfInstance.Load(confmap.Provider(defaultCfgMap, "."), nil); err != nil {
		return fmt.Errorf("error loading hardcoded defaults into koanf: %w", err)
	}

	// Load command-line flags (highest precedence over files and env vars)
	if flags != nil {
		// The posflag.Provider needs the Koanf instance to correctly map flag names to Koanf keys.
		if err := m.koanfInstance.Load(posflag.Provider(flags, ".", m.koanfInstance), nil); err != nil {
			return fmt.Errorf("error loading command-line flags: %w", err)
		}

		debugFlag := flags.Lookup("debug")
		if debugFlag != nil && debugFlag.Value.String() == "true" {
			_ = m.koanfInstance.Set("log.level", "debug")
		}
	}

	// Unmarshal the final merged configuration into m.currentConfig
	var newCfg Config
	if err := m.koanfInstance.UnmarshalWithConf("", &newCfg, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return fmt.Errorf("error unmarshaling final config: %w", err)
	}
	m.currentConfig = newCfg

	// Apply any post-load processing or validation.
	m.postProcessConfig()

	// log.Printf("[DEBUG] Final configuration loaded into Manager. Log Level: %s", m.currentConfig.Log.Level)

	return nil
}

// Get returns a copy of the current configuration.
func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to prevent modification of the internal state.
	// For deep copies, you might need a library or manual copying if structs are complex.
	// For this example, a shallow copy is shown.
	cfgCopy := m.currentConfig
	return cfgCopy
}

// UpdateRuntimeValue updates a specific configuration value at runtime.
// This is a simplified example; a more robust solution would involve:
// - Validating the key and value.
// - Potentially re-unmarshaling or selectively updating m.currentConfig.
// - Notifying other parts of the application about the change (e.g., via an event bus).
func (m *Manager) UpdateRuntimeValue(key string, value interface{}) error {
	return nil
}

// postProcessConfig handles any adjustments needed after loading and unmarshaling.
func (m *Manager) postProcessConfig() {}

// DefaultConfigAsMap converts the DefaultConfig struct to a map[string]interface{}
// for Koanf's confmap.Provider. This is a bit manual but ensures Koanf knows all keys.
func DefaultConfigAsMap() map[string]interface{} {
	def := DefaultConfig()
	// This can be done more elegantly with reflection or a library if the struct is very large.
	return map[string]interface{}{
		// Log configuration
		"log.level":  def.Log.Level,
		"log.format": def.Log.Format,
		"log.file":   def.Log.File,

		// Server configuration
		"server.addr":          def.Server.Addr,
		"server.port":          def.Server.Port,
		"server.ui_enabled":    def.Server.UIEnabled,
		"server.api_enabled":   def.Server.APIEnabled,
		"server.jobs_enabled":  def.Server.JobsEnabled,
		"server.workspace_dir": def.Server.WorkspaceDir,
		"server.concurrency":   def.Server.Concurrency,
		"server.read_timeout":  def.Server.ReadTimeout,
		"server.write_timeout": def.Server.WriteTimeout,

		// UI configuration
		"server.ui.dev_mode":    def.Server.UI.DevMode,
		"server.ui.assets_path": def.Server.UI.AssetsPath,

		// Auth configuration
		"server.auth.mode":  def.Server.Auth.Mode,
		"server.auth.token": def.Server.Auth.Token,
	}
}

// BindFlags defines command-line flags corresponding to configuration settings.
// These flags allow overriding config file / environment variable settings.
// This function should be called when setting up Cobra commands.
func BindFlags(flags *pflag.FlagSet) {
	// Get default config to provide default values for flags' help text
	// defaults := DefaultConfig()

	// Log flags
	// flags.String("log.level", defaults.Log.Level, "Log level (debug, info, warn, error)")
	// flags.String("log.format", defaults.Log.Format, "Log format (text, json)")
	// flags.String("log.file", defaults.Log.File, "Path to log file (optional, leave empty for stdout)")

	var flagvar bool
	flags.BoolVar(&flagvar, "debug", false, "Enable debug logging")

	// Note: The main --config / -c flag for specifying the config file path
	// is typically defined directly on the root Cobra command's PersistentFlags.
}
