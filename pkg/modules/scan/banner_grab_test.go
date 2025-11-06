// pkg/modules/scan/banner_grab_test.go
package scan

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pentora-ai/pentora/pkg/engine"
)

func mustListenTCP(t *testing.T, addr string) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("skipping test: listening on TCP sockets is not permitted in this environment")
		}
		t.Fatalf("failed to listen on %s: %v", addr, err)
	}
	return ln
}

func mustNewHTTPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	defer func() {
		if r := recover(); r != nil {
			if strings.Contains(fmt.Sprint(r), "operation not permitted") {
				t.Skip("skipping test: unable to start HTTP test server in this environment")
			}
			panic(r)
		}
	}()
	srv = httptest.NewServer(handler)
	return srv
}

func TestNewBannerGrabModule(t *testing.T) {
	t.Parallel()

	module := newBannerGrabModule()
	if module == nil {
		t.Fatal("Expected non-nil module, got nil")
	}
	if module.meta.Name != "banner-grabber" {
		t.Errorf("Expected module name 'banner-grabber', got '%s'", module.meta.Name)
	}
	if module.meta.Type != engine.ScanModuleType {
		t.Errorf("Expected module type '%s', got '%s'", engine.ScanModuleType, module.meta.Type)
	}
	if module.config.ReadTimeout != 10*time.Second {
		t.Errorf("Expected read timeout 10s, got %v", module.config.ReadTimeout)
	}
	if module.config.BufferSize != 2048 {
		t.Errorf("Expected buffer size 2048, got %d", module.config.BufferSize)
	}
}

func TestBannerGrabModule_Init(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      map[string]interface{}
		expected    BannerGrabConfig
		expectError bool
	}{
		{
			name: "default values",
			config: map[string]interface{}{
				"read_timeout":             "4s",
				"connect_timeout":          "3s",
				"tls_insecure_skip_verify": false,
			},
			expected: BannerGrabConfig{
				ReadTimeout:           4 * time.Second,
				ConnectTimeout:        3 * time.Second,
				BufferSize:            2048,
				Concurrency:           50,
				SendProbes:            true,
				TLSInsecureSkipVerify: false,
			},
		},
		{
			name: "invalid timeout values",
			config: map[string]interface{}{
				"read_timeout":    "invalid",
				"connect_timeout": "invalid",
			},
			expected: BannerGrabConfig{
				ReadTimeout:           10 * time.Second,
				ConnectTimeout:        5 * time.Second,
				BufferSize:            2048,
				Concurrency:           50,
				SendProbes:            true,
				TLSInsecureSkipVerify: true, // Phase 1.6: Default to true for service detection
			},
			expectError: false,
		},
		{
			name: "custom buffer size and concurrency",
			config: map[string]interface{}{
				"buffer_size": 4096,
				"concurrency": 100,
			},
			expected: BannerGrabConfig{
				ReadTimeout:           10 * time.Second,
				ConnectTimeout:        5 * time.Second,
				BufferSize:            4096,
				Concurrency:           100,
				SendProbes:            true,
				TLSInsecureSkipVerify: true, // Phase 1.6: Default to true for service detection
			},
		},
		{
			name: "disable probes",
			config: map[string]interface{}{
				"send_probes": false,
			},
			expected: BannerGrabConfig{
				ReadTimeout:           10 * time.Second,
				ConnectTimeout:        5 * time.Second,
				BufferSize:            2048,
				Concurrency:           50,
				SendProbes:            false,
				TLSInsecureSkipVerify: true, // Phase 1.6: Default to true for service detection
			},
		},
		{
			name: "invalid sanitize values",
			config: map[string]interface{}{
				"read_timeout":    "0s",
				"connect_timeout": "0s",
				"buffer_size":     -1,
				"concurrency":     -1,
			},
			expected: BannerGrabConfig{
				ReadTimeout:           10 * time.Second,
				ConnectTimeout:        5 * time.Second,
				BufferSize:            2048,
				Concurrency:           1,
				SendProbes:            true,
				TLSInsecureSkipVerify: true, // Phase 1.6: Default to true for service detection
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			module := newBannerGrabModule()
			err := module.Init("instanceId", tt.config)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if module.config.ReadTimeout != tt.expected.ReadTimeout {
				t.Errorf("Expected ReadTimeout %v, got %v", tt.expected.ReadTimeout, module.config.ReadTimeout)
			}
			if module.config.ConnectTimeout != tt.expected.ConnectTimeout {
				t.Errorf("Expected ConnectTimeout %v, got %v", tt.expected.ConnectTimeout, module.config.ConnectTimeout)
			}
			if module.config.BufferSize != tt.expected.BufferSize {
				t.Errorf("Expected BufferSize %d, got %d", tt.expected.BufferSize, module.config.BufferSize)
			}
			if module.config.Concurrency != tt.expected.Concurrency {
				t.Errorf("Expected Concurrency %d, got %d", tt.expected.Concurrency, module.config.Concurrency)
			}
			if module.config.SendProbes != tt.expected.SendProbes {
				t.Errorf("Expected SendProbes %v, got %v", tt.expected.SendProbes, module.config.SendProbes)
			}
			if module.config.TLSInsecureSkipVerify != tt.expected.TLSInsecureSkipVerify {
				t.Errorf("Expected TLSInsecureSkipVerify %v, got %v", tt.expected.TLSInsecureSkipVerify, module.config.TLSInsecureSkipVerify)
			}
		})
	}
}

func TestRunProbesCollectsHTTPEvidence(t *testing.T) {
	t.Parallel()

	ln := listenOnPreferredPort(t, []int{8080, 8000, 8008, 8443})
	defer func() { _ = ln.Close() }()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Server", "PentoraTest/1.0")
			_, _ = fmt.Fprint(w, "hello from test")
		}),
	}

	go func() {
		_ = server.Serve(ln)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = server.Shutdown(ctx)
		cancel()
	}()

	host := "127.0.0.1"
	port := ln.Addr().(*net.TCPAddr).Port

	module := newBannerGrabModule()
	module.config.SendProbes = true
	module.config.ConnectTimeout = 500 * time.Millisecond
	module.config.ReadTimeout = 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result := module.runProbes(ctx, host, port)
	if result.Banner == "" || !strings.Contains(result.Banner, "HTTP/1.1") {
		t.Fatalf("expected HTTP banner in result, got %q", result.Banner)
	}

	if result.IsTLS {
		t.Fatalf("expected non-TLS HTTP probe, got TLS=true")
	}

	foundHTTPProbe := false
	for _, ev := range result.Evidence {
		if ev.ProbeID == "http-get" {
			foundHTTPProbe = true
			if !strings.Contains(ev.Response, "HTTP/1.1") {
				t.Fatalf("expected HTTP response in probe evidence, got %q", ev.Response)
			}
		}
	}

	if !foundHTTPProbe {
		t.Fatalf("expected http-get probe in evidence")
	}
}

func listenOnPreferredPort(t *testing.T, ports []int) net.Listener {
	t.Helper()
	var lastErr error
	for _, p := range ports {
		addr := fmt.Sprintf("127.0.0.1:%d", p)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln
		}
		lastErr = err
	}
	if lastErr != nil {
		t.Skipf("no available test port for HTTP probes: %v", lastErr)
	}
	return mustListenTCP(t, "127.0.0.1:0")
}

func TestBannerGrabModule_Execute_MissingInput(t *testing.T) {
	t.Parallel()

	module := newBannerGrabModule()
	outputChan := make(chan engine.ModuleOutput, 1)

	err := module.Execute(context.Background(), map[string]interface{}{}, outputChan)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	select {
	case output := <-outputChan:
		if output.FromModuleName != "banner-grab-instance" {
			t.Errorf("Expected FromModuleName 'banner-grab-instance', got '%s'", output.FromModuleName)
		}
		if output.DataKey != "service.banner.tcp" {
			t.Errorf("Expected DataKey 'service.banner.tcp', got '%s'", output.DataKey)
		}
		if results, ok := output.Data.([]BannerGrabResult); !ok {
			t.Errorf("Expected output.Data to be of type []BannerGrabResult")
		} else if len(results) != 0 {
			t.Errorf("Expected no banner results, got %d", len(results))
		}
	default:
		t.Error("Expected warning output but got none")
	}
}

func TestGrabGenericBanner(t *testing.T) {
	t.Parallel()

	ln := mustListenTCP(t, "127.0.0.1:0")
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = conn.Write([]byte("TEST BANNER\n"))
	}()

	module := newBannerGrabModule()
	module.config.ReadTimeout = 1 * time.Second
	module.config.ConnectTimeout = 1 * time.Second

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split host/port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("atoi: %v", err)
	}

	banner, _, err := module.grabGenericBanner(context.Background(), host, port)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(banner, "TEST BANNER") {
		t.Errorf("Expected banner to contain 'TEST BANNER', got: %s", banner)
	}
}

func TestGrabGenericBanner_Timeout(t *testing.T) {
	t.Parallel()

	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "TEST BANNER")
	})

	server := mustNewHTTPServer(t, handlerFunc)
	defer server.Close()

	addr := server.URL[len("http://"):]
	// host, portStr, _ := net.SplitHostPort(addr)
	// port, _ := strconv.Atoi(portStr)

	module := newBannerGrabModule()
	module.config.ReadTimeout = 300 * time.Millisecond
	module.config.ConnectTimeout = 300 * time.Millisecond

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host/port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("atoi: %v", err)
	}

	_, _, err = module.grabGenericBanner(context.Background(), host, port)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected error to contain 'timeout', got: %v", err)
	}
}

func TestBannerGrabModule_GrabGenericBanner_Err(t *testing.T) {
	t.Parallel()

	module := newBannerGrabModule()
	module.config.ReadTimeout = 1 * time.Second
	module.config.ConnectTimeout = 1 * time.Second

	_, _, err := module.grabGenericBanner(context.Background(), "invalid-hostname", 65000)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestBannerGrabModuleFactory(t *testing.T) {
	t.Parallel()

	factory := BannerGrabModuleFactory()
	if factory == nil {
		t.Fatal("Expected non-nil factory, got nil")
	}
	_, ok := factory.(*BannerGrabModule)
	if !ok {
		t.Error("Expected factory to return *BannerGrabModule")
	}
}

func TestBannerGrabModule_Metadata(t *testing.T) {
	t.Parallel()

	module := newBannerGrabModule()
	meta := module.Metadata()

	if meta.ID != "banner-grab-instance" {
		t.Errorf("Expected ID 'banner-grab-instance', got '%s'", meta.ID)
	}
	if meta.Name != "banner-grabber" {
		t.Errorf("Expected Name 'banner-grabber', got '%s'", meta.Name)
	}
	if meta.Version != "0.1.0" {
		t.Errorf("Expected Version '0.1.0', got '%s'", meta.Version)
	}
	if meta.Description == "" {
		t.Error("Expected non-empty Description")
	}
	if meta.Type != engine.ScanModuleType {
		t.Errorf("Expected Type '%s', got '%s'", engine.ScanModuleType, meta.Type)
	}
	if meta.Author == "" {
		t.Error("Expected non-empty Author")
	}
	if len(meta.Produces) == 0 || meta.Produces[0].Key != "service.banner.tcp" {
		t.Errorf("Expected Produces to contain 'service.banner.tcp', got %v", meta.Produces)
	}
	if len(meta.Consumes) == 0 || meta.Consumes[0].Key != "discovery.open_tcp_ports" {
		t.Errorf("Expected Consumes to contain 'discovery.open_tcp_ports', got %v", meta.Consumes)
	}
	if meta.ConfigSchema == nil {
		t.Error("Expected non-nil ConfigSchema")
	}
}
