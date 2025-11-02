package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRegisterCommonSchema_Idempotent verifies that RegisterCommonSchema
// can be called multiple times without errors.
func TestRegisterCommonSchema_Idempotent(t *testing.T) {
	dc := NewDataContext()

	// Call multiple times
	RegisterCommonSchema(dc)
	RegisterCommonSchema(dc)
	RegisterCommonSchema(dc)

	// Verify config.targets schema exists
	_, ok := dc.schema["config.targets"]
	require.True(t, ok, "config.targets schema should be registered")

	// Verify config.ports schema exists
	_, ok = dc.schema["config.ports"]
	require.True(t, ok, "config.ports schema should be registered")
}

// TestRegisterCommonSchema_ConfigKeys tests config key schemas.
func TestRegisterCommonSchema_ConfigKeys(t *testing.T) {
	dc := NewDataContext()
	RegisterCommonSchema(dc)

	// Test config.targets ([]string, single)
	err := Publish(dc, "config.targets", []string{"192.168.1.0/24", "10.0.0.1"})
	require.NoError(t, err)

	targets, err := Get[[]string](dc, "config.targets")
	require.NoError(t, err)
	require.Equal(t, []string{"192.168.1.0/24", "10.0.0.1"}, targets)

	// Test config.ports ([]int, single)
	err = Publish(dc, "config.ports", []int{22, 80, 443})
	require.NoError(t, err)

	ports, err := Get[[]int](dc, "config.ports")
	require.NoError(t, err)
	require.Equal(t, []int{22, 80, 443}, ports)
}

// TestRegisterCommonSchema_SimpleParseKeys tests primitive parse key schemas.
func TestRegisterCommonSchema_SimpleParseKeys(t *testing.T) {
	dc := NewDataContext()
	RegisterCommonSchema(dc)

	tests := []struct {
		key   string
		value interface{}
	}{
		{"ssh.banner", "SSH-2.0-OpenSSH_8.9p1"},
		{"ssh.version", "8.9p1"},
		{"http.server", "nginx/1.21.6"},
		{"tls.version", "TLSv1.3"},
		{"service.port", 22},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			// Append values (list cardinality)
			switch v := tt.value.(type) {
			case string:
				err := Append(dc, tt.key, v)
				require.NoError(t, err)
				err = Append(dc, tt.key, v+"_2")
				require.NoError(t, err)

				result, err := Get[[]string](dc, tt.key)
				require.NoError(t, err)
				require.Len(t, result, 2)
				require.Equal(t, v, result[0])
			case int:
				err := Append(dc, tt.key, v)
				require.NoError(t, err)
				err = Append(dc, tt.key, v+1)
				require.NoError(t, err)

				result, err := Get[[]int](dc, tt.key)
				require.NoError(t, err)
				require.Len(t, result, 2)
				require.Equal(t, v, result[0])
			}
		})
	}
}

// TestRegisterCommonSchema_TypeMismatchRejected verifies that type mismatches
// are caught for registered keys.
func TestRegisterCommonSchema_TypeMismatchRejected(t *testing.T) {
	dc := NewDataContext()
	RegisterCommonSchema(dc)

	// Try to publish wrong type for config.targets (expects []string)
	err := dc.PublishValue("config.targets", "single-string")
	require.Error(t, err)
	require.Contains(t, err.Error(), "type mismatch")

	// Try to append wrong type for ssh.banner (expects string)
	err = dc.AppendValue("ssh.banner", 123)
	require.Error(t, err)
	require.Contains(t, err.Error(), "type mismatch")
}

// TestRegisterCommonSchema_LegacyFallback verifies that unregistered keys
// still work via legacy paths.
func TestRegisterCommonSchema_LegacyFallback(t *testing.T) {
	dc := NewDataContext()
	RegisterCommonSchema(dc)

	// Use an unregistered key
	dc.SetInitial("custom.unregistered.key", "value1")
	dc.AddOrAppendToList("custom.unregistered.key", "value2")

	// Should work via legacy API
	all := dc.GetAll()
	require.Contains(t, all, "custom.unregistered.key")

	got, ok := dc.Get("custom.unregistered.key")
	require.True(t, ok)
	require.Equal(t, []interface{}{"value1", "value2"}, got)
}
