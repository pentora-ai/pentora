// pkg/engine/registry_test.go
package engine

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// --- Mock Module for Testing ---
type MockTestModuleConfig struct {
	TestValue string
}

type MockTestModule struct {
	meta   ModuleMetadata
	config MockTestModuleConfig
	inited bool
}

func NewMockTestModule() Module { // Factory signature
	return &MockTestModule{
		meta: ModuleMetadata{
			ID:          "mock-test-module-instance", // Instance ID, name can be from factory
			Name:        "mock-test-module",
			Version:     "1.0",
			Description: "A mock module for testing.",
			Type:        "test",
			Produces:    []DataContractEntry{{Key: "test.output"}},
		},
	}
}

func (m *MockTestModule) Metadata() ModuleMetadata {
	return m.meta
}

func (m *MockTestModule) Init(instanceID string, configMap map[string]interface{}) error {
	// Simple config parsing for test
	if val, ok := configMap["TestValue"].(string); ok {
		m.config.TestValue = val
	} else {
		return fmt.Errorf("missing or invalid TestValue in config")
	}
	m.inited = true
	fmt.Printf("MockTestModule initialized with TestValue: %s\n", m.config.TestValue)
	return nil
}

func (m *MockTestModule) Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- ModuleOutput) error {
	if !m.inited {
		return fmt.Errorf("module not initialized")
	}
	outputChan <- ModuleOutput{
		FromModuleName: m.meta.ID,
		DataKey:        m.meta.Produces[0].Key,
		Data:           fmt.Sprintf("Executed with config: %s", m.config.TestValue),
		Timestamp:      time.Now(),
	}
	return nil
}

// --- End Mock Module ---

func resetRegistry() {
	// Helper to reset registry for isolated tests
	moduleRegistry = make(map[string]ModuleFactory)
}

func TestRegisterModuleFactory(t *testing.T) {
	resetRegistry()
	moduleName := "test-module-1"
	RegisterModuleFactory(moduleName, NewMockTestModule)

	if _, exists := moduleRegistry[moduleName]; !exists {
		t.Errorf("Module factory for '%s' was not registered.", moduleName)
	}

	// Test overwriting (optional, based on desired behavior)
	RegisterModuleFactory(moduleName, NewMockTestModule) // Registering again
	if len(moduleRegistry) != 1 {
		t.Errorf("Expected registry size to be 1 after re-registering, got %d", len(moduleRegistry))
	}
}

func TestGetModuleInstance_Success(t *testing.T) {
	resetRegistry()
	moduleName := "test-module-success"
	RegisterModuleFactory(moduleName, NewMockTestModule)

	config := map[string]interface{}{"TestValue": "hello"}
	instance, err := GetModuleInstance("", moduleName, config)
	if err != nil {
		t.Fatalf("GetModuleInstance failed: %v", err)
	}
	if instance == nil {
		t.Fatal("GetModuleInstance returned a nil instance.")
	}

	// Check if Init was called (via our mock's inited field)
	if mockInstance, ok := instance.(*MockTestModule); ok {
		if !mockInstance.inited {
			t.Error("Expected module Init to be called, but it wasn't.")
		}
		if mockInstance.config.TestValue != "hello" {
			t.Errorf("Expected config TestValue to be 'hello', got '%s'", mockInstance.config.TestValue)
		}
	} else {
		t.Fatal("Instance is not of type *MockTestModule")
	}

	if instance.Metadata().Name != "mock-test-module" {
		t.Errorf("Expected module name 'mock-test-module', got '%s'", instance.Metadata().Name)
	}
}

func TestGetModuleInstance_NotFound(t *testing.T) {
	resetRegistry()
	config := map[string]interface{}{"TestValue": "world"}
	_, err := GetModuleInstance("", "non-existent-module", config)

	if err == nil {
		t.Fatal("Expected error for non-existent module, got nil.")
	}
	expectedErrorMsg := "no module factory registered for name: non-existent-module"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestGetModuleInstance_InitFailure(t *testing.T) {
	resetRegistry()
	moduleName := "test-module-init-fail"
	RegisterModuleFactory(moduleName, NewMockTestModule)

	configMissingValue := map[string]interface{}{} // Missing TestValue
	_, err := GetModuleInstance("", moduleName, configMissingValue)

	if err == nil {
		t.Fatal("Expected error from module Init, got nil.")
	}
	// The error message will be wrapped, so check for a substring
	// if !strings.Contains(err.Error(), "missing or invalid TestValue in config") {
	//  t.Errorf("Expected error to contain 'missing or invalid TestValue', got '%v'", err)
	// }
	// More precise check based on the wrapped error from GetModuleInstance
	expectedErrorMsgPart := "failed to initialize module 'test-module-init-fail': missing or invalid TestValue in config"
	if err.Error() != expectedErrorMsgPart {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsgPart, err.Error())
	}
}
