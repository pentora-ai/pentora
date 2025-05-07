package scanner

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestScanPortClosed(t *testing.T) {
	result := ScanPort("127.0.0.1", 65000) // unlikely to be open
	if result {
		t.Errorf("expected port 65000 to be closed on localhost")
	}
}

func TestScanPortOpen(t *testing.T) {
	// Create a temporary TCP listener on a random available port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open temporary listener: %v", err)
	}
	defer func() {
		if err := ln.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	result := ScanPort("127.0.0.1", port)
	if !result {
		t.Errorf("expected port %d to be reported as open", port)
	}
}

func TestGrabBanner(t *testing.T) {
	bannerText := "SSH-2.0-TestBanner\r\n"

	// Start a temporary TCP server that sends a banner
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open test TCP listener: %v", err)
	}
	defer func() {
		if err := ln.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	go func() {
		conn, err := ln.Accept()
		if err == nil {
			_, _ = conn.Write([]byte(bannerText))
			_ = conn.Close()
		}
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	banner, err := GrabBanner("127.0.0.1", port)
	if err != nil {
		t.Errorf("unexpected error from GrabBanner: %v", err)
	}
	if !strings.Contains(banner, "SSH-2.0-TestBanner") {
		t.Errorf("expected banner to contain %q, got %q", bannerText, banner)
	}

	// Small delay to ensure go routine exits cleanly
	time.Sleep(50 * time.Millisecond)
}

func TestHTTPProbe(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "Pentora-Test-HTTP")
		_, _ = fmt.Fprintln(w, "Hello from Pentora")
	}))
	defer ts.Close()

	host := strings.TrimPrefix(ts.URL, "http://")
	parts := strings.Split(host, ":")
	ip := parts[0]
	port := 80

	_, err := fmt.Sscanf(parts[1], "%d", &port)
	if err != nil {
		t.Fatalf("port parsing failed: %v", err)
	}

	banner, err := HTTPProbe(ip, port)
	if err != nil {
		t.Fatalf("unexpected error from HTTPProbe: %v", err)
	}
	if !strings.Contains(banner, "Pentora-Test-HTTP") {
		t.Errorf("expected banner to contain 'Pentora-Test-HTTP', got %q", banner)
	}
}
