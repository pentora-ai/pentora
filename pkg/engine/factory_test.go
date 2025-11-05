package engine

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestDefaultAppManagerFactory_Create(t *testing.T) {
	factory := &DefaultAppManagerFactory{}

	t.Run("Create with empty config", func(t *testing.T) {
		manager, err := factory.Create(nil, "")
		assert.NoError(t, err)
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.ctx)
		assert.NotNil(t, manager.cancel)
		assert.NotNil(t, manager.ConfigManager)
		assert.NotNil(t, manager.EventManager)
		assert.NotNil(t, manager.HookManager)
	})
}

func TestDefaultAppManagerFactory_CreateWithNoConfig(t *testing.T) {
	factory := &DefaultAppManagerFactory{}

	manager, err := factory.CreateWithNoConfig()
	assert.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestDefaultAppManagerFactory_GetRuntimeLogLevel(t *testing.T) {
	factory := &DefaultAppManagerFactory{}

	tests := []struct {
		name          string
		verbosityFlag int
		expectedLevel zerolog.Level
	}{
		{"No flags", 0, zerolog.WarnLevel},
		{"Verbosity 1", 1, zerolog.InfoLevel},
		{"Verbosity 2", 2, zerolog.DebugLevel},
		{"Verbosity 3", 3, zerolog.TraceLevel},
		{"Invalid verbosity", 4, zerolog.WarnLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flags.CountP("verbosity", "v", "verbosity level")
			for i := 0; i < tt.verbosityFlag; i++ {
				flags.Parse([]string{"-v"})
			}

			level := factory.GetRuntimeLogLevel(flags)
			assert.Equal(t, tt.expectedLevel, level)
		})
	}

	t.Run("Nil flags", func(t *testing.T) {
		level := factory.GetRuntimeLogLevel(nil)
		assert.Equal(t, zerolog.DebugLevel, level)
	})
}

func TestDefaultAppManagerFactory_CreateWithConfig(t *testing.T) {
	factory := &DefaultAppManagerFactory{}

	t.Run("CreateWithConfig with empty config", func(t *testing.T) {
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		manager, err := factory.CreateWithConfig(flags, "")
		assert.NoError(t, err)
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.ctx)
		assert.NotNil(t, manager.cancel)
		assert.NotNil(t, manager.ConfigManager)
		assert.NotNil(t, manager.EventManager)
		assert.NotNil(t, manager.HookManager)
	})

	t.Run("CreateWithConfig with nil flags", func(t *testing.T) {
		manager, err := factory.CreateWithConfig(nil, "")
		assert.NoError(t, err)
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.ConfigManager)
	})
}
