package config

import (
	"sync"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// Helper to reset global variables for testing
func resetGlobalConfig() {
	k = nil
	once = sync.Once{}
}

func TestInitGlobalConfig_InitializesKoanfOnce(t *testing.T) {
	resetGlobalConfig()
	InitGlobalConfig()
	assert.NotNil(t, k, "Global koanf instance should be initialized")
}

func TestInitGlobalConfig_IsIdempotent(t *testing.T) {
	resetGlobalConfig()
	InitGlobalConfig()
	firstInstance := k
	InitGlobalConfig()
	secondInstance := k
	assert.Equal(t, firstInstance, secondInstance, "Koanf instance should not change on repeated InitGlobalConfig calls")
}

func TestInitGlobalConfig_KoanfUsesDotDelimiter(t *testing.T) {
	resetGlobalConfig()
	InitGlobalConfig()
	assert.Equal(t, ".", k.Delim(), "Koanf delimiter should be '.'")
}

func TestNewManager_InitializesManagerWithGlobalKoanf(t *testing.T) {
	resetGlobalConfig()
	manager := NewManager()
	assert.NotNil(t, manager, "Manager should not be nil")
	assert.NotNil(t, manager.koanfInstance, "Manager's koanfInstance should not be nil")
	assert.Equal(t, k, manager.koanfInstance, "Manager's koanfInstance should use the global Koanf instance")
}

func TestNewManager_GlobalKoanfIsInitialized(t *testing.T) {
	resetGlobalConfig()
	_ = NewManager()
	assert.NotNil(t, k, "Global Koanf instance should be initialized by NewManager")
}

func TestNewManager_MultipleManagersShareGlobalKoanf(t *testing.T) {
	resetGlobalConfig()
	manager1 := NewManager()
	manager2 := NewManager()
	assert.Equal(t, manager1.koanfInstance, manager2.koanfInstance, "All managers should share the same global Koanf instance")
}

func TestDefaultConfig_ReturnsExpectedDefaults(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "info", cfg.Log.Level, "Default log level should be 'info'")
	assert.Equal(t, "text", cfg.Log.Format, "Default log format should be 'text'")
	assert.Equal(t, "", cfg.Log.File, "Default log file should be empty")
}

func TestManager_Load_LoadsDefaultsWhenNoFlags(t *testing.T) {
	resetGlobalConfig()
	manager := NewManager()
	err := manager.Load(nil, "")
	assert.NoError(t, err, "Load should not return error when loading defaults")
	cfg := manager.Get()
	assert.Equal(t, "info", cfg.Log.Level, "Default log level should be 'info'")
	assert.Equal(t, "text", cfg.Log.Format, "Default log format should be 'text'")
	assert.Equal(t, "", cfg.Log.File, "Default log file should be empty")
}

func TestManager_Load_OverridesWithFlags(t *testing.T) {
	resetGlobalConfig()
	manager := NewManager()
	flags := newTestFlagSet()
	_ = flags.Set("log.level", "error")
	_ = flags.Set("log.format", "json")
	_ = flags.Set("log.file", "/tmp/test.log")
	err := manager.Load(flags, "")
	assert.NoError(t, err, "Load should not return error when loading with flags")
	cfg := manager.Get()
	assert.Equal(t, "error", cfg.Log.Level, "Flag should override log level")
	assert.Equal(t, "json", cfg.Log.Format, "Flag should override log format")
	assert.Equal(t, "/tmp/test.log", cfg.Log.File, "Flag should override log file")
}

func TestManager_Load_DebugFlagSetsLogLevelToDebug(t *testing.T) {
	resetGlobalConfig()
	manager := NewManager()
	flags := newTestFlagSet()
	_ = flags.Set("debug", "true")
	err := manager.Load(flags, "")
	assert.NoError(t, err, "Load should not return error when loading with debug flag")
	cfg := manager.Get()
	assert.Equal(t, "debug", cfg.Log.Level, "Debug flag should set log level to debug")
}

func TestBindFlags_AddsDebugFlag(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindFlags(flags)
	debugFlag := flags.Lookup("debug")
	assert.NotNil(t, debugFlag, "BindFlags should add a 'debug' flag")
	assert.Equal(t, "Enable debug logging", debugFlag.Usage, "Debug flag should have correct usage")
	assert.Equal(t, "false", debugFlag.DefValue, "Debug flag should default to false")
}

func TestBindFlags_DebugFlagDefaultValue(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindFlags(flags)
	val, err := flags.GetBool("debug")
	assert.NoError(t, err, "Should be able to get 'debug' flag value")
	assert.False(t, val, "Default value of 'debug' flag should be false")
}

func TestBindFlags_DebugFlagCanBeSet(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindFlags(flags)
	err := flags.Set("debug", "true")
	assert.NoError(t, err, "Should be able to set 'debug' flag")
	val, err := flags.GetBool("debug")
	assert.NoError(t, err, "Should be able to get 'debug' flag value after setting")
	assert.True(t, val, "Value of 'debug' flag should be true after setting")
}

func TestManager_UpdateRuntimeValue_NoOpReturnsNil(t *testing.T) {
	resetGlobalConfig()
	manager := NewManager()
	err := manager.UpdateRuntimeValue("log.level", "warn")
	assert.NoError(t, err, "UpdateRuntimeValue should return nil (no error) for any input")
}

func TestManager_UpdateRuntimeValue_DoesNotChangeConfig(t *testing.T) {
	resetGlobalConfig()
	manager := NewManager()
	_ = manager.Load(nil, "")
	originalCfg := manager.Get()

	_ = manager.UpdateRuntimeValue("log.level", "warn")
	afterCfg := manager.Get()

	assert.Equal(t, originalCfg, afterCfg, "UpdateRuntimeValue should not modify config (no-op)")
}

func newTestFlagSet() *pflag.FlagSet {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("log.level", "info", "")
	flags.String("log.format", "text", "")
	flags.String("log.file", "", "")
	flags.Bool("debug", false, "")
	return flags
}
