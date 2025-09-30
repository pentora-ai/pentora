package reporting

import (
	"context"
	"testing"
	"time"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/modules/discovery"
	"github.com/pentora-ai/pentora/pkg/modules/parse"
	"github.com/pentora-ai/pentora/pkg/modules/scan"
)

func TestAssetProfileBuilderUsesFingerprintForNonDefaultPort(t *testing.T) {
	module := newAssetProfileBuilderModule()
	if err := module.Init(assetProfileBuilderModuleTypeName, map[string]interface{}{}); err != nil {
		t.Fatalf("init module failed: %v", err)
	}

	target := "192.0.2.10"
	port := 20022

	inputs := map[string]interface{}{
		"config.targets": []string{target},
		"discovery.open_tcp_ports": []interface{}{
			discovery.TCPPortDiscoveryResult{Target: target, OpenPorts: []int{port}},
		},
		"service.banner.tcp": []interface{}{
			scan.BannerGrabResult{IP: target, Port: port, Banner: "SSH-2.0-OpenSSH_8.9p1"},
		},
		"service.fingerprint.details": []interface{}{
			parse.FingerprintParsedInfo{Target: target, Port: port, Protocol: "ssh", Product: "OpenSSH", Version: "8.9p1", Confidence: 0.92},
		},
	}

	outputChan := make(chan engine.ModuleOutput, 1)
	if err := module.Execute(context.Background(), inputs, outputChan); err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	select {
	case out := <-outputChan:
		profiles, ok := out.Data.([]engine.AssetProfile)
		if !ok {
			t.Fatalf("expected []engine.AssetProfile, got %T", out.Data)
		}
		if len(profiles) == 0 {
			t.Fatalf("no asset profiles returned")
		}
		profile := profiles[0]
		ports := profile.OpenPorts[target]
		if len(ports) == 0 {
			t.Fatalf("expected open port entry")
		}
		service := ports[0].Service
		if service.Name != "ssh" {
			t.Fatalf("expected service name ssh, got %s", service.Name)
		}
		if service.Product != "OpenSSH" {
			t.Fatalf("expected product OpenSSH, got %s", service.Product)
		}
		if service.Version != "8.9p1" {
			t.Fatalf("expected version 8.9p1, got %s", service.Version)
		}
	case <-time.After(time.Second):
		t.Fatal("no output emitted")
	}
}
