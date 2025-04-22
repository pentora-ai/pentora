package parser

import (
	"testing"
)

func TestHTTPBannerParse(t *testing.T) {
	banner := "HTTP/1.1 200 OK\r\nServer: nginx/1.18.0\r\nContent-Type: text/html\r\n\r\n"
	info := Dispatch(banner)

	if info == nil {
		t.Fatal("Expected ServiceInfo, got nil")
	}
	if info.Name != "nginx" {
		t.Errorf("Expected name 'nginx', got %s", info.Name)
	}
	if info.Version != "1.18.0" {
		t.Errorf("Expected version '1.18.0', got %s", info.Version)
	}
}
