package parse

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/fingerprint"
	"github.com/pentora-ai/pentora/pkg/modules/scan"
)

// --- MOCK RESOLVER ---

type mockResolver struct {
	resolveFn func(ctx context.Context, input fingerprint.Input) (fingerprint.Result, error)
}

func (m mockResolver) Resolve(ctx context.Context, input fingerprint.Input) (fingerprint.Result, error) {
	return m.resolveFn(ctx, input)
}

// --- TESTS ---

func TestFingerprintParserModule_Execute_FullCoverage(t *testing.T) {
	// Arrange
	originalGetResolver := getResolver
	defer func() { getResolver = originalGetResolver }()

	calledInputs := []fingerprint.Input{}

	getResolver = func() fingerprint.Resolver {
		return mockResolver{
			resolveFn: func(ctx context.Context, input fingerprint.Input) (fingerprint.Result, error) {
				calledInputs = append(calledInputs, input)

				switch {
				case strings.Contains(input.Banner, "error"):
					return fingerprint.Result{}, errors.New("resolver error")
				case strings.Contains(input.Banner, "unknown"):
					return fingerprint.Result{}, nil
				default:
					return fingerprint.Result{
						Product:     "TestProduct",
						Vendor:      "TestVendor",
						Version:     "1.0",
						CPE:         "cpe:/a:test:product:1.0",
						Confidence:  0.9,
						Description: "Test Description",
					}, nil
				}
			},
		}
	}

	m := newFingerprintParserModule()
	_ = m.Init("test-instance", nil)

	banner := scan.BannerGrabResult{
		IP:       "127.0.0.1",
		Port:     22,
		Protocol: "tcp",
		Banner:   "SSH-2.0-OpenSSH_8.9",
		Evidence: []scan.ProbeObservation{
			{Response: "HTTP/1.1 200 OK\r\nServer: nginx", Protocol: "http", ProbeID: "probe1"},
			{Response: "error-banner", Protocol: "http", ProbeID: "probe2"},
			{Response: "unknown-banner", Protocol: "ftp", ProbeID: "probe3"},
			{Response: "HTTP/1.1 200 OK\r\nServer: nginx", Protocol: "http", ProbeID: "probe1"}, // duplicate
		},
	}

	inputs := map[string]interface{}{
		"service.banner.tcp": []interface{}{banner},
	}
	outputChan := make(chan engine.ModuleOutput, 10)
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel() // ctx.Done() branch
	}()

	err := m.Execute(ctx, inputs, outputChan)

	// Assert
	if err != nil && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}

	close(outputChan)

	count := 0
	for out := range outputChan {
		parsed, ok := out.Data.(FingerprintParsedInfo)
		if !ok {
			t.Errorf("output type mismatch: %T", out.Data)
			continue
		}
		if parsed.Product != "TestProduct" {
			t.Errorf("unexpected product: %v", parsed.Product)
		}
		count++
	}

	if count == 0 {
		t.Error("expected at least one parsed fingerprint result")
	}
}

func TestFingerprintParserModule_Execute_NoInputKey(t *testing.T) {
	m := newFingerprintParserModule()
	out := make(chan engine.ModuleOutput)
	defer close(out)

	err := m.Execute(context.Background(), map[string]interface{}{}, out)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestFingerprintParserModule_Execute_InvalidType(t *testing.T) {
	m := newFingerprintParserModule()
	out := make(chan engine.ModuleOutput)
	defer close(out)

	err := m.Execute(context.Background(), map[string]interface{}{
		"service.banner.tcp": "not-a-list",
	}, out)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestFingerprintParserModule_Execute_InvalidElementType(t *testing.T) {
	m := newFingerprintParserModule()
	out := make(chan engine.ModuleOutput)
	defer close(out)

	err := m.Execute(context.Background(), map[string]interface{}{
		"service.banner.tcp": []interface{}{"not-banner"},
	}, out)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestFingerprintParserModule_fingerprintProtocolHint_AllBranches(t *testing.T) {
	cases := map[string]string{
		"SSH-2.0":                    "ssh",
		"HTTP/1.1 200 OK":            "http",
		"Server: nginx":              "http",
		"EHLO smtp.gmail.com":        "smtp",
		"ftp ready":                  "ftp",
		"MySQL server version 8.0.1": "mysql",
		"unknown banner":             "",
	}

	for banner, want := range cases {
		got := fingerprintProtocolHint(0, banner)
		if got != want {
			t.Errorf("banner %q => got %q, want %q", banner, got, want)
		}
	}
}

func TestFingerprintParserModule_gatherBannerCandidates(t *testing.T) {
	banner := scan.BannerGrabResult{
		Banner:   "HTTP/1.1 200 OK",
		Protocol: "tcp",
		Evidence: []scan.ProbeObservation{
			{Response: "SSH-2.0-OpenSSH_8.9", Protocol: "", ProbeID: "probe1"},
			{Response: "   ", Protocol: "http", ProbeID: "probe2"}, // boş response skip
		},
	}

	candidates := gatherBannerCandidates(banner)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	if candidates[0].ProbeID != "tcp-passive" {
		t.Errorf("expected first ProbeID 'tcp-passive', got %s", candidates[0].ProbeID)
	}
	if candidates[1].Protocol != "tcp" {
		t.Errorf("expected inherited protocol 'tcp', got %s", candidates[1].Protocol)
	}
}

func TestFingerprintParserModule_Metadata(t *testing.T) {
	m := newFingerprintParserModule()
	meta := m.Metadata()

	if meta.ID != fingerprintParserModuleID {
		t.Errorf("unexpected ID: %v", meta.ID)
	}
	if meta.Name != fingerprintParserModuleName {
		t.Errorf("unexpected Name: %v", meta.Name)
	}
	if len(meta.Consumes) == 0 || len(meta.Produces) == 0 {
		t.Error("expected consumes/produces metadata to be set")
	}
}

func TestFingerprintParserModule_Init(t *testing.T) {
	m := newFingerprintParserModule()
	err := m.Init("test-id", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if m.meta.ID != "test-id" {
		t.Errorf("expected meta.ID = 'test-id', got %s", m.meta.ID)
	}
}

func TestFingerprintParserModule_fingerprintParserModuleFactory(t *testing.T) {
	mod := fingerprintParserModuleFactory()
	if mod == nil {
		t.Fatal("expected non-nil module")
	}
	meta := mod.Metadata()
	if meta.Name != fingerprintParserModuleName {
		t.Errorf("unexpected factory module name: %s", meta.Name)
	}
}
