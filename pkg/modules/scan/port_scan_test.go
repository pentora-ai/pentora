package scan

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/pentora-ai/pentora/pkg/engine"
)

func TestPortScanModule_Metadata(t *testing.T) {
	module := newPortScanModule()
	meta := module.Metadata()

	expected := engine.ModuleMetadata{
		ID:          "tcp-connect-scan-instance",
		Name:        "tcp-connect-scan",
		Version:     "0.1.0",
		Description: "Performs TCP connect scans to identify open ports on target hosts.",
		Type:        engine.ScanModuleType,
		Author:      "Pentora Team",
		Tags:        []string{"scan", "port", "tcp", "connect"},
		Produces:    []string{"scan.port_status"},
		Consumes:    []string{"discovery.live_hosts", "config.targets_override"},
		ConfigSchema: map[string]engine.ParameterDefinition{
			"targets":     {Description: "Optional override for target IPs/CIDRs. Usually consumed from 'discovery.live_hosts'.", Type: "[]string", Required: false},
			"ports":       {Description: "Comma-separated list of ports and/or ranges (e.g., '22,80,1000-1024').", Type: "string", Required: false, Default: "21,22,23,25,53,80,110,143,443,445,3306,3389,5432,5900,8080,8443"},
			"timeout":     {Description: "Timeout for each port connection attempt (e.g., '1s').", Type: "duration", Required: false, Default: "1s"},
			"concurrency": {Description: "Number of concurrent port scanning goroutines per host.", Type: "int", Required: false, Default: 100},
			"scan_type":   {Description: "Type of scan to perform (currently only 'tcp_connect').", Type: "string", Required: false, Default: "tcp_connect"},
		},
	}

	// Only check fields that are not maps/slices for direct equality
	if meta.ID != expected.ID ||
		meta.Name != expected.Name ||
		meta.Version != expected.Version ||
		meta.Description != expected.Description ||
		meta.Type != expected.Type ||
		meta.Author != expected.Author {
		t.Errorf("Metadata() returned unexpected basic fields: got %+v, want %+v", meta, expected)
	}

	if !reflect.DeepEqual(meta.Tags, expected.Tags) {
		t.Errorf("Metadata() Tags = %v, want %v", meta.Tags, expected.Tags)
	}
	if !reflect.DeepEqual(meta.Produces, expected.Produces) {
		t.Errorf("Metadata() Produces = %v, want %v", meta.Produces, expected.Produces)
	}
	if !reflect.DeepEqual(meta.Consumes, expected.Consumes) {
		t.Errorf("Metadata() Consumes = %v, want %v", meta.Consumes, expected.Consumes)
	}
	if !reflect.DeepEqual(meta.ConfigSchema, expected.ConfigSchema) {
		t.Errorf("Metadata() ConfigSchema = %+v, want %+v", meta.ConfigSchema, expected.ConfigSchema)
	}
}
func TestPortScanModule_Init(t *testing.T) {
	tests := []struct {
		name       string
		input      map[string]interface{}
		wantConfig PortScanConfig
	}{
		{
			name:  "Empty config uses defaults",
			input: map[string]interface{}{},
			wantConfig: PortScanConfig{
				Targets:     nil,
				Ports:       "21,22,23,25,53,80,110,143,443,445,3306,3389,5432,5900,8080,8443",
				Timeout:     1 * time.Second,
				Concurrency: 100,
				ScanType:    "tcp_connect",
			},
		},
		{
			name: "Override all fields",
			input: map[string]interface{}{
				"targets":     []string{"1.2.3.4", "5.6.7.8"},
				"ports":       "80,443,8080-8082",
				"timeout":     "3s",
				"concurrency": 42,
				"scan_type":   "tcp_connect",
			},
			wantConfig: PortScanConfig{
				Targets:     []string{"1.2.3.4", "5.6.7.8"},
				Ports:       "80,443,8080-8082",
				Timeout:     3 * time.Second,
				Concurrency: 42,
				ScanType:    "tcp_connect",
			},
		},
		{
			name: "Invalid timeout falls back to default",
			input: map[string]interface{}{
				"timeout": "notaduration",
			},
			wantConfig: PortScanConfig{
				Targets:     nil,
				Ports:       "21,22,23,25,53,80,110,143,443,445,3306,3389,5432,5900,8080,8443",
				Timeout:     1 * time.Second,
				Concurrency: 100,
				ScanType:    "tcp_connect",
			},
		},
		{
			name: "Invalid scan_type falls back to default",
			input: map[string]interface{}{
				"scan_type": "udp_scan",
			},
			wantConfig: PortScanConfig{
				Targets:     nil,
				Ports:       "21,22,23,25,53,80,110,143,443,445,3306,3389,5432,5900,8080,8443",
				Timeout:     1 * time.Second,
				Concurrency: 100,
				ScanType:    "tcp_connect",
			},
		},
		{
			name: "Concurrency less than 1 is set to 1",
			input: map[string]interface{}{
				"concurrency": 0,
			},
			wantConfig: PortScanConfig{
				Targets:     nil,
				Ports:       "21,22,23,25,53,80,110,143,443,445,3306,3389,5432,5900,8080,8443",
				Timeout:     1 * time.Second,
				Concurrency: 1,
				ScanType:    "tcp_connect",
			},
		},
		{
			name: "Timeout less than or equal to 0 is set to 1s",
			input: map[string]interface{}{
				"timeout": "0s",
			},
			wantConfig: PortScanConfig{
				Targets:     nil,
				Ports:       "21,22,23,25,53,80,110,143,443,445,3306,3389,5432,5900,8080,8443",
				Timeout:     1 * time.Second,
				Concurrency: 100,
				ScanType:    "tcp_connect",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module := newPortScanModule()
			err := module.Init(tt.input)
			if err != nil {
				t.Errorf("Init() returned error: %v", err)
			}
			got := module.config

			if !reflect.DeepEqual(got.Targets, tt.wantConfig.Targets) {
				t.Errorf("Targets = %v, want %v", got.Targets, tt.wantConfig.Targets)
			}
			if got.Ports != tt.wantConfig.Ports {
				t.Errorf("Ports = %v, want %v", got.Ports, tt.wantConfig.Ports)
			}
			if got.Timeout != tt.wantConfig.Timeout {
				t.Errorf("Timeout = %v, want %v", got.Timeout, tt.wantConfig.Timeout)
			}
			if got.Concurrency != tt.wantConfig.Concurrency {
				t.Errorf("Concurrency = %v, want %v", got.Concurrency, tt.wantConfig.Concurrency)
			}
			if got.ScanType != tt.wantConfig.ScanType {
				t.Errorf("ScanType = %v, want %v", got.ScanType, tt.wantConfig.ScanType)
			}
		})
	}
}
func TestPortScanModule_Execute_NoTargets(t *testing.T) {
	module := newPortScanModule()
	outputChan := make(chan engine.ModuleOutput, 1)
	ctx := context.Background()

	// No targets in config or input
	err := module.Execute(ctx, map[string]interface{}{}, outputChan)
	if err == nil {
		t.Errorf("Expected error when no targets are provided")
	}
	select {
	case out := <-outputChan:
		if out.Error == nil {
			t.Errorf("Expected output error when no targets, got nil")
		}
	default:
		t.Errorf("Expected output on channel when no targets")
	}
}

func TestPortScanModule_Execute_InvalidPorts(t *testing.T) {
	module := newPortScanModule()
	module.config.Targets = []string{"127.0.0.1"}
	module.config.Ports = "invalid-port"
	outputChan := make(chan engine.ModuleOutput, 1)
	ctx := context.Background()

	err := module.Execute(ctx, map[string]interface{}{}, outputChan)
	if err == nil {
		t.Errorf("Expected error for invalid ports string")
	}
	select {
	case out := <-outputChan:
		if out.Error == nil {
			t.Errorf("Expected output error for invalid ports, got nil")
		}
	default:
		t.Errorf("Expected output on channel for invalid ports")
	}
}

func TestPortScanModule_Execute_EmptyPorts(t *testing.T) {
	module := newPortScanModule()
	module.config.Targets = []string{"127.0.0.1"}
	module.config.Ports = ""
	outputChan := make(chan engine.ModuleOutput, 1)
	ctx := context.Background()

	err := module.Execute(ctx, map[string]interface{}{}, outputChan)
	if err == nil {
		t.Errorf("Expected error for empty ports string")
	}
	select {
	case out := <-outputChan:
		if out.Error == nil {
			t.Errorf("Expected output error for empty ports, got nil")
		}
	default:
		t.Errorf("Expected output on channel for empty ports")
	}
}

func TestPortScanModule_Execute_SingleTargetSinglePort(t *testing.T) {
	module := newPortScanModule()
	module.config.Targets = []string{"127.0.0.1"}
	module.config.Ports = "65535" // Unlikely to be open
	module.config.Concurrency = 2
	module.config.Timeout = 500 * time.Millisecond

	outputChan := make(chan engine.ModuleOutput, 2)
	ctx := context.Background()

	err := module.Execute(ctx, map[string]interface{}{}, outputChan)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	found := false
	for i := 0; i < 1; i++ {
		select {
		case out := <-outputChan:
			status, ok := out.Data.(PortStatusInfo)
			if !ok {
				t.Errorf("Output Data is not PortStatusInfo: %T", out.Data)
			}
			if status.IP != "127.0.0.1" || status.Port != 65535 {
				t.Errorf("Unexpected scan result: %+v", status)
			}
			if status.Status != "closed" && status.Status != "filtered" {
				t.Errorf("Expected status closed or filtered, got %s", status.Status)
			}
			found = true
		case <-time.After(2 * time.Second):
			t.Errorf("Timeout waiting for output")
		}
	}
	if !found {
		t.Errorf("Did not receive expected output")
	}
}

func TestPortScanModule_Execute_LiveHostsInput(t *testing.T) {
	module := newPortScanModule()
	module.config.Ports = "65534"
	module.config.Concurrency = 1
	module.config.Timeout = 500 * time.Millisecond

	outputChan := make(chan engine.ModuleOutput, 2)
	ctx := context.Background()

	// Simulate discovery.live_hosts as []string
	inputs := map[string]interface{}{
		"discovery.live_hosts": []string{"127.0.0.1"},
	}
	err := module.Execute(ctx, inputs, outputChan)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	select {
	case out := <-outputChan:
		status, ok := out.Data.(PortStatusInfo)
		if !ok {
			t.Errorf("Output Data is not PortStatusInfo: %T", out.Data)
		}
		if status.IP != "127.0.0.1" || status.Port != 65534 {
			t.Errorf("Unexpected scan result: %+v", status)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("Timeout waiting for output")
	}
}

func TestPortScanModule_Execute_ContextCancel(t *testing.T) {
	module := newPortScanModule()
	module.config.Targets = []string{"127.0.0.1"}
	module.config.Ports = "80,81,82,83,84,85"
	module.config.Concurrency = 1
	module.config.Timeout = 2 * time.Second

	outputChan := make(chan engine.ModuleOutput, 10)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	err := module.Execute(ctx, map[string]interface{}{}, outputChan)
	if err != context.Canceled && err != nil {
		t.Errorf("Expected context.Canceled or nil, got: %v", err)
	}
}
func TestPortScanModuleFactory_ReturnsNewInstance(t *testing.T) {
	module1 := PortScanModuleFactory()
	module2 := PortScanModuleFactory()

	if module1 == nil {
		t.Errorf("PortScanModuleFactory() returned nil")
	}
	if module2 == nil {
		t.Errorf("PortScanModuleFactory() returned nil on second call")
	}
	if module1 == module2 {
		t.Errorf("PortScanModuleFactory() returned the same instance on multiple calls; expected new instance each time")
	}

	// Check type assertion
	_, ok := module1.(*PortScanModule)
	if !ok {
		t.Errorf("PortScanModuleFactory() did not return *PortScanModule, got %T", module1)
	}
}

func TestPortScanModule_Execute_PrioritizesInputOverConfig(t *testing.T) {
	module := newPortScanModule()
	module.config.Targets = []string{"10.0.0.1"}
	module.config.Ports = "1"
	module.config.Concurrency = 1
	module.config.Timeout = 500 * time.Millisecond

	outputChan := make(chan engine.ModuleOutput, 1)
	ctx := context.Background()

	inputs := map[string]interface{}{
		"discovery.live_hosts": []string{"127.0.0.1"},
	}
	err := module.Execute(ctx, inputs, outputChan)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	select {
	case out := <-outputChan:
		data := out.Data.(PortStatusInfo)
		if data.IP != "127.0.0.1" {
			t.Errorf("Expected input IP 127.0.0.1, got %s", data.IP)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestPortScanModule_Execute_InvalidInputTypeFallbackToConfig(t *testing.T) {
	module := newPortScanModule()
	module.config.Targets = []string{"127.0.0.1"}
	module.config.Ports = "1"
	module.config.Concurrency = 1
	module.config.Timeout = 500 * time.Millisecond

	inputs := map[string]interface{}{
		"discovery.live_hosts": 1234, // invalid type
	}
	out := make(chan engine.ModuleOutput, 1)
	err := module.Execute(context.Background(), inputs, out)
	if err != nil {
		t.Errorf("expected fallback to config, got error: %v", err)
	}
}
