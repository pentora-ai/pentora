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

func TestGetRegisteredModuleFactories(t *testing.T) {
	resetRegistry()

	// Test with empty registry
	factories := GetRegisteredModuleFactories()
	if len(factories) != 0 {
		t.Errorf("Expected empty registry, got %d entries", len(factories))
	}

	// Register some test modules
	module1 := "test-module-1"
	module2 := "test-module-2"
	RegisterModuleFactory(module1, NewMockTestModule)
	RegisterModuleFactory(module2, NewMockTestModule)

	// Get copy of registry
	factories = GetRegisteredModuleFactories()

	// Verify correct number of entries
	if len(factories) != 2 {
		t.Errorf("Expected 2 factories, got %d", len(factories))
	}

	// Verify expected modules exist
	if _, exists := factories[module1]; !exists {
		t.Errorf("Expected factory '%s' not found in registry copy", module1)
	}
	if _, exists := factories[module2]; !exists {
		t.Errorf("Expected factory '%s' not found in registry copy", module2)
	}

	// Verify copy is independent - modify copy should not affect original
	delete(factories, module1)
	originalFactories := GetRegisteredModuleFactories()
	if len(originalFactories) != 2 {
		t.Error("Original registry was modified when modifying copy")
	}
}

func TestGetAllModuleMetadata_EmptyRegistry(t *testing.T) {
	resetRegistry()
	metadata, err := GetAllModuleMetadata()
	if err != nil {
		t.Fatalf("Expected no error for empty registry, got: %v", err)
	}
	if len(metadata) != 0 {
		t.Errorf("Expected empty metadata slice, got %d entries", len(metadata))
	}
}

func TestGetAllModuleMetadata_SingleModule(t *testing.T) {
	resetRegistry()
	moduleName := "mock-test-module"
	RegisterModuleFactory(moduleName, NewMockTestModule)

	metadata, err := GetAllModuleMetadata()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(metadata) != 1 {
		t.Fatalf("Expected 1 metadata entry, got %d", len(metadata))
	}
	meta := metadata[0]
	if meta.Name != moduleName {
		t.Errorf("Expected metadata Name '%s', got '%s'", moduleName, meta.Name)
	}
	if meta.Description != "A mock module for testing." {
		t.Errorf("Unexpected Description: %s", meta.Description)
	}
}

func TestGetAllModuleMetadata_MultipleModules(t *testing.T) {
	resetRegistry()
	module1 := "mock-module-1"
	module2 := "mock-module-2"
	RegisterModuleFactory(module1, NewMockTestModule)
	RegisterModuleFactory(module2, NewMockTestModule)

	metadata, err := GetAllModuleMetadata()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(metadata) != 2 {
		t.Fatalf("Expected 2 metadata entries, got %d", len(metadata))
	}
	names := map[string]bool{}
	for _, meta := range metadata {
		names[meta.Name] = true
	}
	if !names[module1] || !names[module2] {
		t.Errorf("Expected metadata for both modules, got: %v", names)
	}
}

func TestGetAllModuleMetadata_FactoryReturnsNil(t *testing.T) {
	resetRegistry()
	badFactory := func() Module { return nil }
	RegisterModuleFactory("bad-module", badFactory)

	_, err := GetAllModuleMetadata()
	if err == nil {
		t.Fatal("Expected error when factory returns nil, got nil")
	}
	expected := "module factory for 'bad-module' returned a nil instance"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestGetAllModuleMetadata_MetadataNameMismatch(t *testing.T) {
	resetRegistry()
	// Factory returns module with empty Name
	mismatchFactory := func() Module {
		return &MockTestModule{
			meta: ModuleMetadata{
				ID:          "instance-id",
				Name:        "",
				Version:     "1.0",
				Description: "Mismatch name",
				Type:        "test",
				Produces:    []DataContractEntry{{Key: "output"}},
			},
		}
	}
	moduleName := "mismatch-module"
	RegisterModuleFactory(moduleName, mismatchFactory)

	metadata, err := GetAllModuleMetadata()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(metadata) != 1 {
		t.Fatalf("Expected 1 metadata entry, got %d", len(metadata))
	}
	if metadata[0].Name != moduleName {
		t.Errorf("Expected metadata Name to be set to registered name '%s', got '%s'", moduleName, metadata[0].Name)
	}
}
