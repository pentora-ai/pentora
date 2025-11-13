package fingerprint

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockFingerprinter is a test implementation of Fingerprinter interface
type mockFingerprinter struct {
	id        string
	priority  Priority
	protocols []string
}

func (m *mockFingerprinter) ID() string                   { return m.id }
func (m *mockFingerprinter) Priority() Priority           { return m.priority }
func (m *mockFingerprinter) SupportedProtocols() []string { return m.protocols }
func (m *mockFingerprinter) AnalyzePassive(ctx context.Context, obs PassiveObservation) (*ServiceCandidate, bool, error) {
	return nil, false, nil
}
func (m *mockFingerprinter) ActiveProbes() []Probe { return nil }
func (m *mockFingerprinter) Verify(ctx context.Context, probe Probe, response []byte) (*ServiceCandidate, bool, error) {
	return nil, false, nil
}

// resetRegistry clears the registry for test isolation
func resetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	fingerprinters = make(map[string]Fingerprinter)
	protocolIndex = make(map[string][]string)
}

func TestRegisterFingerprinter_ValidNamespace(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		namespace string
	}{
		{"builtin namespace", "builtin.ssh", "builtin"},
		{"extended namespace", "extended.http-advanced", "extended"},
		{"custom namespace", "custom.my-protocol", "custom"},
		{"plugin namespace", "plugin.vendor-scanner", "plugin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRegistry()

			fp := &mockFingerprinter{
				id:        tt.id,
				priority:  PriorityBuiltin,
				protocols: []string{"test"},
			}

			// Should not panic
			require.NotPanics(t, func() {
				RegisterFingerprinter(fp)
			})

			// Verify registration
			registered := fingerprinters[tt.id]
			require.NotNil(t, registered)
			require.Equal(t, tt.id, registered.ID())
		})
	}
}

func TestRegisterFingerprinter_InvalidNamespace(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"no namespace", "ssh"},
		{"wrong separator", "builtin-ssh"},
		{"invalid prefix", "invalid.ssh"},
		{"empty id", ""},
		{"only namespace", "builtin."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRegistry()

			fp := &mockFingerprinter{
				id:        tt.id,
				priority:  PriorityBuiltin,
				protocols: []string{"test"},
			}

			// Should panic with error message
			require.Panics(t, func() {
				RegisterFingerprinter(fp)
			})
		})
	}
}

func TestRegisterFingerprinter_DuplicateID(t *testing.T) {
	resetRegistry()

	fp1 := &mockFingerprinter{
		id:        "builtin.ssh",
		priority:  PriorityBuiltin,
		protocols: []string{"ssh"},
	}

	fp2 := &mockFingerprinter{
		id:        "builtin.ssh", // Duplicate
		priority:  PriorityBuiltin,
		protocols: []string{"ssh"},
	}

	// First registration should succeed
	require.NotPanics(t, func() {
		RegisterFingerprinter(fp1)
	})

	// Second registration with same ID should panic
	require.Panics(t, func() {
		RegisterFingerprinter(fp2)
	})
}

func TestRegisterFingerprinter_CoreProtocolProtection(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		protocol    string
		shouldPanic bool
		setupFunc   func() // Setup function to register builtin first
	}{
		{
			name:        "extended cannot shadow builtin ssh",
			id:          "extended.ssh",
			protocol:    "ssh",
			shouldPanic: true,
			setupFunc: func() {
				RegisterFingerprinter(&mockFingerprinter{
					id:        "builtin.ssh",
					priority:  PriorityBuiltin,
					protocols: []string{"ssh"},
				})
			},
		},
		{
			name:        "extended cannot shadow builtin http",
			id:          "extended.http",
			protocol:    "http",
			shouldPanic: true,
			setupFunc: func() {
				RegisterFingerprinter(&mockFingerprinter{
					id:        "builtin.http",
					priority:  PriorityBuiltin,
					protocols: []string{"http"},
				})
			},
		},
		{
			name:        "custom can override core protocol",
			id:          "custom.ssh",
			protocol:    "ssh",
			shouldPanic: false,
			setupFunc: func() {
				RegisterFingerprinter(&mockFingerprinter{
					id:        "builtin.ssh",
					priority:  PriorityBuiltin,
					protocols: []string{"ssh"},
				})
			},
		},
		{
			name:        "plugin can override core protocol",
			id:          "plugin.ssh",
			protocol:    "ssh",
			shouldPanic: false,
			setupFunc: func() {
				RegisterFingerprinter(&mockFingerprinter{
					id:        "builtin.ssh",
					priority:  PriorityBuiltin,
					protocols: []string{"ssh"},
				})
			},
		},
		{
			name:        "extended allowed for non-core protocol",
			id:          "extended.redis",
			protocol:    "redis",
			shouldPanic: false,
			setupFunc:   func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRegistry()

			// Setup
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			fp := &mockFingerprinter{
				id:        tt.id,
				priority:  PriorityExtended,
				protocols: []string{tt.protocol},
			}

			if tt.shouldPanic {
				require.Panics(t, func() {
					RegisterFingerprinter(fp)
				})
			} else {
				require.NotPanics(t, func() {
					RegisterFingerprinter(fp)
				})
			}
		})
	}
}

func TestGetFingerprinter_Priority(t *testing.T) {
	resetRegistry()

	// Register multiple implementations with different priorities
	RegisterFingerprinter(&mockFingerprinter{
		id:        "builtin.ssh",
		priority:  PriorityBuiltin,
		protocols: []string{"ssh"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "custom.ssh",
		priority:  PriorityCustom,
		protocols: []string{"ssh"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "plugin.ssh",
		priority:  PriorityPlugin,
		protocols: []string{"ssh"},
	})

	// Should return highest priority (plugin)
	fp := GetFingerprinter("ssh")
	require.NotNil(t, fp)
	require.Equal(t, "plugin.ssh", fp.ID())
	require.Equal(t, PriorityPlugin, fp.Priority())
}

func TestGetFingerprinter_NotFound(t *testing.T) {
	resetRegistry()

	fp := GetFingerprinter("nonexistent")
	require.Nil(t, fp)
}

func TestGetFingerprintersByProtocol(t *testing.T) {
	resetRegistry()

	// Register multiple implementations
	RegisterFingerprinter(&mockFingerprinter{
		id:        "builtin.ssh",
		priority:  PriorityBuiltin,
		protocols: []string{"ssh"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "custom.ssh",
		priority:  PriorityCustom,
		protocols: []string{"ssh"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "plugin.ssh",
		priority:  PriorityPlugin,
		protocols: []string{"ssh"},
	})

	fps := GetFingerprintersByProtocol("ssh")
	require.Len(t, fps, 3)

	// Verify sorted by priority (highest first)
	require.Equal(t, "plugin.ssh", fps[0].ID())
	require.Equal(t, "custom.ssh", fps[1].ID())
	require.Equal(t, "builtin.ssh", fps[2].ID())
}

func TestGetFingerprintersByProtocol_NotFound(t *testing.T) {
	resetRegistry()

	fps := GetFingerprintersByProtocol("nonexistent")
	require.Nil(t, fps)
}

func TestListFingerprinters(t *testing.T) {
	resetRegistry()

	// Register multiple fingerprinters
	RegisterFingerprinter(&mockFingerprinter{
		id:        "builtin.ssh",
		priority:  PriorityBuiltin,
		protocols: []string{"ssh"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "builtin.http",
		priority:  PriorityBuiltin,
		protocols: []string{"http"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "custom.ssh",
		priority:  PriorityCustom,
		protocols: []string{"ssh"},
	})

	fps := ListFingerprinters()
	require.Len(t, fps, 3)

	// Verify all registered fingerprinters are present
	ids := make([]string, len(fps))
	for i, fp := range fps {
		ids[i] = fp.ID()
	}
	sort.Strings(ids)

	expected := []string{"builtin.http", "builtin.ssh", "custom.ssh"}
	require.Equal(t, expected, ids)
}

func TestGetRegistryStats(t *testing.T) {
	resetRegistry()

	// Register fingerprinters across namespaces and protocols
	RegisterFingerprinter(&mockFingerprinter{
		id:        "builtin.ssh",
		priority:  PriorityBuiltin,
		protocols: []string{"ssh"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "builtin.http",
		priority:  PriorityBuiltin,
		protocols: []string{"http"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "extended.redis",
		priority:  PriorityExtended,
		protocols: []string{"redis"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "custom.ssh",
		priority:  PriorityCustom,
		protocols: []string{"ssh"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "plugin.http",
		priority:  PriorityPlugin,
		protocols: []string{"http"},
	})

	stats := GetRegistryStats()

	// Verify totals
	require.Equal(t, 5, stats.Total)

	// Verify namespace counts
	require.Equal(t, 2, stats.ByNamespace["builtin"])
	require.Equal(t, 1, stats.ByNamespace["extended"])
	require.Equal(t, 1, stats.ByNamespace["custom"])
	require.Equal(t, 1, stats.ByNamespace["plugin"])

	// Verify protocol counts
	require.Equal(t, 2, stats.ByProtocol["ssh"])
	require.Equal(t, 2, stats.ByProtocol["http"])
	require.Equal(t, 1, stats.ByProtocol["redis"])

	// Verify metadata list
	require.Len(t, stats.Fingerprinters, 5)

	// Verify sorting (highest priority first, then by ID)
	require.Equal(t, "plugin.http", stats.Fingerprinters[0].ID)
	require.Equal(t, PriorityPlugin, stats.Fingerprinters[0].Priority)

	// Find custom.ssh (should be before builtin entries)
	var customIdx, builtinIdx int
	for i, meta := range stats.Fingerprinters {
		if meta.ID == "custom.ssh" {
			customIdx = i
		}
		if meta.ID == "builtin.ssh" {
			builtinIdx = i
		}
	}
	require.Less(t, customIdx, builtinIdx, "custom.ssh should appear before builtin.ssh")
}

func TestExtractProtocol(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{"builtin.ssh", "builtin.ssh", "ssh"},
		{"extended.http", "extended.http", "http"},
		{"custom.redis", "custom.redis", "redis"},
		{"plugin.postgres", "plugin.postgres", "postgres"},
		{"no dot", "invalid", ""},
		{"empty", "", ""},
		{"only namespace", "builtin.", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractProtocol(tt.id)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		wantError bool
	}{
		{"builtin valid", "builtin.ssh", false},
		{"extended valid", "extended.http", false},
		{"custom valid", "custom.redis", false},
		{"plugin valid", "plugin.postgres", false},
		{"invalid prefix", "invalid.ssh", true},
		{"no dot", "builtin", true},
		{"empty", "", true},
		{"wrong separator", "builtin-ssh", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNamespace(tt.id)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsExtendedOverridingCore(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		protocol  string
		setupFunc func()
		expected  bool
	}{
		{
			name:     "extended.ssh overriding builtin.ssh",
			id:       "extended.ssh",
			protocol: "ssh",
			setupFunc: func() {
				resetRegistry()
				RegisterFingerprinter(&mockFingerprinter{
					id:        "builtin.ssh",
					priority:  PriorityBuiltin,
					protocols: []string{"ssh"},
				})
			},
			expected: true,
		},
		{
			name:     "extended.http overriding builtin.http",
			id:       "extended.http",
			protocol: "http",
			setupFunc: func() {
				resetRegistry()
				RegisterFingerprinter(&mockFingerprinter{
					id:        "builtin.http",
					priority:  PriorityBuiltin,
					protocols: []string{"http"},
				})
			},
			expected: true,
		},
		{
			name:      "custom.ssh not considered overriding",
			id:        "custom.ssh",
			protocol:  "ssh",
			setupFunc: func() { resetRegistry() },
			expected:  false,
		},
		{
			name:      "plugin.ssh not considered overriding",
			id:        "plugin.ssh",
			protocol:  "ssh",
			setupFunc: func() { resetRegistry() },
			expected:  false,
		},
		{
			name:      "extended.redis (non-core) allowed",
			id:        "extended.redis",
			protocol:  "redis",
			setupFunc: func() { resetRegistry() },
			expected:  false,
		},
		{
			name:      "extended.ssh without builtin",
			id:        "extended.ssh",
			protocol:  "ssh",
			setupFunc: func() { resetRegistry() },
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			result := isExtendedOverridingCore(tt.id, tt.protocol)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestNewDefaultCoordinator(t *testing.T) {
	resetRegistry()

	// Register test fingerprinters
	RegisterFingerprinter(&mockFingerprinter{
		id:        "builtin.ssh",
		priority:  PriorityBuiltin,
		protocols: []string{"ssh"},
	})

	RegisterFingerprinter(&mockFingerprinter{
		id:        "builtin.http",
		priority:  PriorityBuiltin,
		protocols: []string{"http"},
	})

	coord := NewDefaultCoordinator()
	require.NotNil(t, coord)
	require.Len(t, coord.fingerprinters, 2)
}

func TestConcurrentRegistration(t *testing.T) {
	resetRegistry()

	// Test concurrent registration doesn't cause races
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			fp := &mockFingerprinter{
				id:        "builtin.test" + string(rune('0'+id)),
				priority:  PriorityBuiltin,
				protocols: []string{"test"},
			}

			RegisterFingerprinter(fp)
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	require.Equal(t, 10, len(fingerprinters))
}

func TestConcurrentRead(t *testing.T) {
	resetRegistry()

	// Register some fingerprinters
	RegisterFingerprinter(&mockFingerprinter{
		id:        "builtin.ssh",
		priority:  PriorityBuiltin,
		protocols: []string{"ssh"},
	})

	done := make(chan bool, 100)

	// Test concurrent reads
	for i := 0; i < 100; i++ {
		go func() {
			defer func() { done <- true }()

			_ = GetFingerprinter("ssh")
			_ = GetFingerprintersByProtocol("ssh")
			_ = ListFingerprinters()
			_ = GetRegistryStats()
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}
