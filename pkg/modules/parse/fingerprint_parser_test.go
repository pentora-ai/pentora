package parse

import (
	"testing"

	"github.com/pentora-ai/pentora/pkg/modules/scan"
)

func TestGatherBannerCandidates(t *testing.T) {
	banner := scan.BannerGrabResult{
		Protocol: "tcp",
		Banner:   " SSH-2.0-OpenSSH_8.2 ",
		Evidence: []scan.ProbeObservation{
			{ProbeID: "http-get", Protocol: "http", Response: " HTTP/1.1 200 OK \r\nServer: Test\r\n\r\n"},
			{ProbeID: "redis-ping", Protocol: "redis", Response: "+PONG\r\n"},
		},
	}

	candidates := gatherBannerCandidates(banner)

	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(candidates))
	}

	if candidates[0].ProbeID != "tcp-passive" {
		t.Fatalf("expected passive candidate first, got %s", candidates[0].ProbeID)
	}
	if candidates[0].Response != "SSH-2.0-OpenSSH_8.2" {
		t.Fatalf("unexpected passive response: %q", candidates[0].Response)
	}

	seen := map[string]bool{}
	for _, c := range candidates {
		if c.Response == "" {
			t.Fatalf("candidate %s has empty response", c.ProbeID)
		}
		seen[c.ProbeID] = true
	}

	if !seen["http-get"] || !seen["redis-ping"] {
		t.Fatalf("expected http-get and redis-ping candidates, got %#v", seen)
	}
}
