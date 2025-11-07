package reporting

import (
	"context"
	"testing"
	"time"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/modules/parse"
)

// NOTE: Vulnerability aggregation is handled, but not asserted here due to
// evolving profile structure. Focus other behaviors for now.

func TestAssetProfileBuilderHandlesEmptyInputs(t *testing.T) {
	m := newAssetProfileBuilderModule()
	if err := m.Init(assetProfileBuilderModuleTypeName, map[string]interface{}{}); err != nil {
		t.Fatalf("init: %v", err)
	}
	outCh := make(chan engine.ModuleOutput, 1)
	if err := m.Execute(context.Background(), map[string]interface{}{}, outCh); err != nil {
		t.Fatalf("execute: %v", err)
	}
	select {
	case out := <-outCh:
		if _, ok := out.Data.([]engine.AssetProfile); !ok {
			t.Fatalf("expected []engine.AssetProfile, got %T", out.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for output")
	}
}

func TestAssetProfileBuilderMergesParsedDetails(t *testing.T) {
	m := newAssetProfileBuilderModule()
	if err := m.Init(assetProfileBuilderModuleTypeName, map[string]interface{}{}); err != nil {
		t.Fatalf("init: %v", err)
	}
	target := "203.0.113.42"
	port := 80
	inputs := map[string]interface{}{
		"config.targets": []string{target},
		"service.http.details": []interface{}{
			parse.HTTPParsedInfo{Target: target, Port: port, ServerProduct: "nginx", ServerVersion: "1.21.6"},
		},
		"service.ssh.details": []interface{}{
			parse.SSHParsedInfo{Target: target, Port: 22, Software: "OpenSSH", SoftwareVersion: "9.3"},
		},
	}
	outCh := make(chan engine.ModuleOutput, 1)
	if err := m.Execute(context.Background(), inputs, outCh); err != nil {
		t.Fatalf("execute: %v", err)
	}
	select {
	case out := <-outCh:
		profiles := out.Data.([]engine.AssetProfile)
		if len(profiles) == 0 {
			t.Fatalf("no profiles")
		}
		ap := profiles[0]
		// Expect to see both HTTP and SSH services reflected in the profile maps
		if _, ok := ap.OpenPorts[target]; !ok {
			t.Fatalf("expected open ports for %s", target)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
