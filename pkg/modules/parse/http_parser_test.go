package parse

import (
	"context"
	"io"
	"testing"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/modules/scan"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.New(io.Discard))
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// --- TESTS ---

func TestHTTPParserModule_Metadata_Init(t *testing.T) {
	m := newHTTPParserModule()
	meta := m.Metadata()
	if meta.Name != httpParserModuleTypeName {
		t.Errorf("unexpected module name: %s", meta.Name)
	}
	err := m.Init("http-test-instance", map[string]interface{}{"dummy": "config"})
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if m.meta.ID != "http-test-instance" {
		t.Errorf("expected ID = http-test-instance, got %s", m.meta.ID)
	}
}

func TestHTTPParserModule_determineScheme_AllBranches(t *testing.T) {
	cases := []struct {
		port   int
		isTLS  bool
		expect string
	}{
		{80, false, "http"},
		{8000, false, "http"},
		{8080, false, "http"},
		{8008, false, "http"},
		{443, false, "https"},
		{8443, false, "https"},
		{1234, true, "https"},
		{1234, false, "http"},
	}
	for _, c := range cases {
		got := determineScheme(c.port, c.isTLS)
		if got != c.expect {
			t.Errorf("port=%d tls=%v: got %s want %s", c.port, c.isTLS, got, c.expect)
		}
	}
}

func TestHTTPParserModule_Execute_NoInputKey(t *testing.T) {
	m := newHTTPParserModule()
	out := make(chan engine.ModuleOutput)
	defer close(out)
	err := m.Execute(context.Background(), map[string]interface{}{}, out)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestHTTPParserModule_Execute_InvalidInputType(t *testing.T) {
	m := newHTTPParserModule()
	out := make(chan engine.ModuleOutput)
	defer close(out)
	err := m.Execute(context.Background(), map[string]interface{}{"service.banner.tcp": 123}, out)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestHTTPParserModule_Execute_ContextCancel(t *testing.T) {
	m := newHTTPParserModule()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel
	out := make(chan engine.ModuleOutput)
	defer close(out)
	inputs := map[string]interface{}{
		"service.banner.tcp": []interface{}{
			scan.BannerGrabResult{Banner: "HTTP/1.1 200 OK\r\n\r\n"},
		},
	}
	err := m.Execute(ctx, inputs, out)
	if err == nil {
		t.Errorf("expected context canceled error")
	}
}

func TestHTTPParserModule_Execute_SkipCases(t *testing.T) {
	m := newHTTPParserModule()
	out := make(chan engine.ModuleOutput, 10)

	// Banners that should be skipped for different reasons
	cases := []scan.BannerGrabResult{
		{Banner: "", Error: "some error"},
		{Banner: "NotHTTP", Error: ""},
		{Banner: "HTTP/1.1 200 OK", IsTLS: true, Port: 443},
	}

	inputs := map[string]interface{}{
		"service.banner.tcp": []interface{}{cases[0], cases[1], cases[2]},
	}
	err := m.Execute(context.Background(), inputs, out)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	close(out)
	if len(out) != 0 {
		t.Errorf("expected no outputs for skipped banners")
	}
}

func TestHTTPParserModule_Execute_ParseAllBranches(t *testing.T) {
	m := newHTTPParserModule()
	out := make(chan engine.ModuleOutput, 20)

	bannerInvalidStatus := "HTTP/1.1\r\nHeader: test\r\n\r\n<body>ignored</body>"
	bannerBadCode := "HTTP/1.1 ABC NotOK\r\nServer: Apache/2.4.1\r\n\r\n"
	bannerGood := "HTTP/1.1 200 OK\r\nServer: nginx/1.18.0\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Length: 50\r\n\r\n<html><head><title>Test Title</title></head><body></body></html>"
	bannerBrokenHeader := "HTTP/1.1 200 OK\r\nServer: Something\r\nContent-Type: text/plain\r\nContent-Length: notanumber\r\n\r\n<body>no title</body>"

	inputs := map[string]interface{}{
		"service.banner.tcp": []interface{}{
			scan.BannerGrabResult{IP: "1.1.1.1", Port: 80, Banner: bannerInvalidStatus},
			scan.BannerGrabResult{IP: "1.1.1.2", Port: 80, Banner: bannerBadCode},
			scan.BannerGrabResult{IP: "1.1.1.3", Port: 8080, Banner: bannerGood},
			scan.BannerGrabResult{IP: "1.1.1.4", Port: 8080, Banner: bannerBrokenHeader},
		},
	}

	err := m.Execute(context.Background(), inputs, out)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	close(out)

	found := 0
	for o := range out {
		data, ok := o.Data.(HTTPParsedInfo)
		if !ok {
			t.Fatalf("unexpected data type: %T", o.Data)
		}
		found++
		if data.Target == "1.1.1.3" {
			if data.HTMLTitle != "Test Title" {
				t.Errorf("expected title parsed, got %q", data.HTMLTitle)
			}
			if data.ServerProduct != "nginx" || data.ServerVersion != "1.18.0" {
				t.Errorf("expected nginx/1.18.0, got %s/%s", data.ServerProduct, data.ServerVersion)
			}
		}
	}
	if found != 4 {
		t.Errorf("expected 4 outputs, got %d", found)
	}
}

func TestHTTPParserModule_Execute_AcceptsTypedSlice(t *testing.T) {
	m := newHTTPParserModule()
	out := make(chan engine.ModuleOutput, 5)
	banners := []scan.BannerGrabResult{
		{IP: "9.9.9.9", Port: 80, Banner: "HTTP/1.0 404 Not Found\r\n\r\n"},
	}
	inputs := map[string]interface{}{
		"service.banner.tcp": banners,
	}
	err := m.Execute(context.Background(), inputs, out)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	close(out)
	got := 0
	for range out {
		got++
	}
	if got != 1 {
		t.Errorf("expected 1 output, got %d", got)
	}
}

func TestHTTPParserModule_Factory(t *testing.T) {
	mod := HTTPParserModuleFactory()
	if mod == nil {
		t.Fatal("expected non-nil module")
	}
	meta := mod.Metadata()
	if meta.Name != httpParserModuleTypeName {
		t.Errorf("unexpected module name: %s", meta.Name)
	}
}
