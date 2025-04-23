package scanner

import (
	"net"
	"testing"
)

func TestDiscoverPortsReturnsOpenPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open test listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	ports := DiscoverPorts("127.0.0.1", port, port)

	found := false
	for _, p := range ports {
		if p == port {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected to discover open port %d, got %v", port, ports)
	}
}

func TestDiscoverPortsReturnsEmptyWhenNoOpenPorts(t *testing.T) {
	ports := DiscoverPorts("127.0.0.1", 65000, 65000) // unlikely to be open
	if len(ports) != 0 {
		t.Errorf("expected no open ports, got %v", ports)
	}
}

func TestItoa(t *testing.T) {
	cases := map[int]string{
		0:    "0",
		1:    "1",
		22:   "22",
		8080: "8080",
		9999: "9999",
	}
	for input, expected := range cases {
		out := itoa(input)
		if out != expected {
			t.Errorf("itoa(%d) = %q; want %q", input, out, expected)
		}
	}
}
