package proto

import "testing"

// Smoke test to ensure generated proto types are present and package compiles under tests.
func TestModuleAPISmoke(t *testing.T) {
	// Instantiate zero-values of a few generated types to ensure linkage.
	var _ ModuleMessage
	var _ HostControlSignal_SignalType
}
