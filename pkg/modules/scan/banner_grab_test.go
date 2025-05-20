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
	if module.config.ReadTimeout != 3*time.Second {
		t.Errorf("Expected read timeout 3s, got %v", module.config.ReadTimeout)
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
				ReadTimeout:           3 * time.Second,
				ConnectTimeout:        2 * time.Second,
				BufferSize:            2048,
				Concurrency:           50,
				SendProbes:            true,
				TLSInsecureSkipVerify: false,
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
				ReadTimeout:           3 * time.Second,
				ConnectTimeout:        2 * time.Second,
				BufferSize:            4096,
				Concurrency:           100,
				SendProbes:            true,
				TLSInsecureSkipVerify: false,
			},
		},
		{
			name: "disable probes",
			config: map[string]interface{}{
				"send_probes": false,
			},
			expected: BannerGrabConfig{
				ReadTimeout:           3 * time.Second,
				ConnectTimeout:        2 * time.Second,
				BufferSize:            2048,
				Concurrency:           50,
				SendProbes:            false,
				TLSInsecureSkipVerify: false,
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
				ReadTimeout:    3 * time.Second,
				ConnectTimeout: 2 * time.Second,
				BufferSize:     2048,
				Concurrency:    1,
				SendProbes:     true,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			module := newBannerGrabModule()
			err := module.Init(tt.config)
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

func TestIsPotentiallyHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		port     int
		expected bool
	}{
		{80, true},
		{443, true},
		{8080, true},
		{22, false},
		{3306, false},
		{8443, true},
		{0, false},
		{65535, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(strconv.Itoa(tt.port), func(t *testing.T) {
			t.Parallel()
			result := isPotentiallyHTTP(tt.port)
			if result != tt.expected {
				t.Errorf("For port %d, expected %v, got %v", tt.port, tt.expected, result)
			}
		})
	}
}

func TestIsPotentiallyTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		port     int
		expected bool
	}{
		{443, true},
		{993, true},
		{995, true},
		{465, true},
		{80, false},
		{8080, false},
		{0, false},
		{65535, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(strconv.Itoa(tt.port), func(t *testing.T) {
			t.Parallel()
			result := isPotentiallyTLS(tt.port)
			if result != tt.expected {
				t.Errorf("For port %d, expected %v, got %v", tt.port, tt.expected, result)
			}
		})
	}
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
		if output.DataKey != "service.banner.raw" {
			t.Errorf("Expected DataKey 'service.banner.raw', got '%s'", output.DataKey)
		}
		if _, ok := output.Data.(BannerGrabResult); !ok {
			t.Errorf("Expected output.Data to be of type BannerGrabResult")
		}
	default:
		t.Error("Expected warning output but got none")
	}
}

func TestBannerGrabModule_Execute_InvalidInputType(t *testing.T) {
	t.Parallel()

	module := newBannerGrabModule()
	outputChan := make(chan engine.ModuleOutput, 1)

	err := module.Execute(context.Background(), map[string]interface{}{
		"scan.port_status": "invalid type",
	}, outputChan)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	select {
	case output := <-outputChan:
		if output.FromModuleName != "banner-grab-instance" {
			t.Errorf("Expected FromModuleName 'banner-grab-instance', got '%s'", output.FromModuleName)
		}
		if output.DataKey != "service.banner.raw" {
			t.Errorf("Expected DataKey 'service.banner.raw', got '%s'", output.DataKey)
		}

		result, ok := output.Data.(BannerGrabResult)
		if !ok {
			t.Fatal("Expected output.Data to be of type BannerGrabResult")
		}
		if result.IP != "unknown" {
			t.Errorf("Expected IP 'unknown', got '%s'", result.IP)
		}
		if result.Error == "" {
			t.Error("Expected error message, got empty string")
		}
	default:
		t.Error("Expected error output but got none")
	}
}

func TestBannerGrabModule_Execute_NonOpenPort(t *testing.T) {
	t.Parallel()

	module := newBannerGrabModule()
	outputChan := make(chan engine.ModuleOutput, 1)

	err := module.Execute(context.Background(), map[string]interface{}{
		"scan.port_status": PortStatusInfo{
			IP:     "127.0.0.1",
			Port:   80,
			Status: "closed",
		},
	}, outputChan)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	select {
	case <-outputChan:
		t.Error("Expected no output for non-open port")
	default:
		// Correct - no output expected
	}
}

func TestGrabGenericBanner(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.Write([]byte("TEST BANNER\n"))
	}()

	module := newBannerGrabModule()
	module.config.ReadTimeout = 1 * time.Second
	module.config.ConnectTimeout = 1 * time.Second

	banner, err := module.grabGenericBanner(context.Background(), ln.Addr().String())
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

	server := httptest.NewServer(handlerFunc)
	defer server.Close()

	addr := server.URL[len("http://"):]
	//host, portStr, _ := net.SplitHostPort(addr)
	//port, _ := strconv.Atoi(portStr)

	module := newBannerGrabModule()
	module.config.ReadTimeout = 300 * time.Millisecond
	module.config.ConnectTimeout = 300 * time.Millisecond

	_, err := module.grabGenericBanner(context.Background(), addr)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected error to contain 'timeout', got: %v", err)
	}
}

func TestGrabHTTPBanner(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			return
		}

		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 12\r\n\r\nHello World!"))
	}()

	module := newBannerGrabModule()
	module.config.ReadTimeout = 1 * time.Second
	module.config.ConnectTimeout = 1 * time.Second

	port := ln.Addr().(*net.TCPAddr).Port
	banner, err := module.grabHTTPBanner(context.Background(), "127.0.0.1", port, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(banner, "HTTP/1.1 200 OK") {
		t.Errorf("Expected banner to contain 'HTTP/1.1 200 OK', got: %s", banner)
	}
	if !strings.Contains(banner, "Hello World!") {
		t.Errorf("Expected banner to contain 'Hello World!', got: %s", banner)
	}
}

func TestGrabTLSBanner(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TLS test in short mode")
	}

	t.Parallel()

	// create a test server that supports TLS
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000")
		w.Header().Set("Custom-Header", "TestValue")
		fmt.Fprintln(w, "Fake HTTPS Server Response")
	}))
	defer server.Close()

	addr := server.URL[len("https://"):]
	module := newBannerGrabModule()
	module.config.ReadTimeout = 2 * time.Second
	module.config.ConnectTimeout = 2 * time.Second
	module.config.TLSInsecureSkipVerify = true

	banner, err := module.grabTLSBanner(context.Background(), addr)
	if err != nil {
		t.Fatalf("Failed to grab TLS banner: %v", err)
	}

	expectedParts := []string{
		"TLSv304; SANs=example.com,*.example.com",
	}

	for _, part := range expectedParts {
		if !strings.Contains(banner, part) {
			t.Errorf("Expected banner to contain %q, got:\n%s", part, banner)
		}
	}
}

func TestBannerGrabModule_Execute_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	module := newBannerGrabModule()
	module.config.ReadTimeout = 1 * time.Second
	module.config.ConnectTimeout = 1 * time.Second
	module.config.SendProbes = true

	outputChan := make(chan engine.ModuleOutput, 1)

	err := module.Execute(context.Background(), map[string]interface{}{
		"scan.port_status": PortStatusInfo{
			IP:     "example.com",
			Port:   80,
			Status: "open",
		},
	}, outputChan)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	select {
	case output := <-outputChan:
		result, ok := output.Data.(BannerGrabResult)
		if !ok {
			t.Fatal("Expected output.Data to be of type BannerGrabResult")
		}
		if result.IP != "example.com" {
			t.Errorf("Expected IP 'example.com', got '%s'", result.IP)
		}
		if result.Port != 80 {
			t.Errorf("Expected port 80, got %d", result.Port)
		}
	default:
		t.Error("Expected output but got none")
	}
}

func TestBannerGrabModule_Execute_IsPotentiallyHTTPButTLS(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:8443")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("BANNER_DATA_TLS"))
	}))
	server.Listener = listener
	server.StartTLS()
	defer server.Close()

	addr := server.URL[len("https://"):]
	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)

	module := newBannerGrabModule()
	module.config.ReadTimeout = 1 * time.Second
	module.config.ConnectTimeout = 1 * time.Second
	module.config.SendProbes = true
	module.config.TLSInsecureSkipVerify = true

	outputChan := make(chan engine.ModuleOutput, 1)
	err = module.Execute(context.Background(), map[string]interface{}{
		"scan.port_status": PortStatusInfo{
			IP:     host,
			Port:   port,
			Status: "open",
		},
	}, outputChan)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	select {
	case output := <-outputChan:
		result, ok := output.Data.(BannerGrabResult)
		if !ok {
			t.Fatal("Expected output.Data to be of type BannerGrabResult")
		}
		if !strings.Contains(result.Banner, "BANNER_DATA_TLS") {
			t.Errorf("Expected banner to not contain 'BANNER_DATA_TLS', got: %s", result.Banner)
		}
	default:
		t.Error("Expected output but got none")
	}
}

func TestBannerGrabModule_Execute_IsPotentiallyTLS(t *testing.T) {
	// TODO: Mock isPotentiallyTLS function
}

func TestBannerGrabModule_GrabGenericBanner_Err(t *testing.T) {
	t.Parallel()

	module := newBannerGrabModule()
	module.config.ReadTimeout = 1 * time.Second
	module.config.ConnectTimeout = 1 * time.Second

	_, err := module.grabGenericBanner(context.Background(), "invalid-address")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedErr := "dial tcp: address invalid-address: missing port in address"
	if expectedErr != err.Error() {
		t.Errorf("Expected error %v, got: %v", expectedErr, err)
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
	if len(meta.Produces) == 0 || meta.Produces[0] != "service.banner.raw" {
		t.Errorf("Expected Produces to contain 'service.banner.raw', got %v", meta.Produces)
	}
	if len(meta.Consumes) == 0 || meta.Consumes[0] != "scan.port_status" {
		t.Errorf("Expected Consumes to contain 'scan.port_status', got %v", meta.Consumes)
	}
	if meta.ConfigSchema == nil {
		t.Error("Expected non-nil ConfigSchema")
	}
}
