package plugin

import (
	"strings"
	"testing"
)

func TestMatchAllReturnsCorrectPlugin(t *testing.T) {
	ctx := map[string]string{"ssh/banner": "SSH-2.0-OpenSSH_7.1p2"}
	openPorts := []int{22}
	satisfied := []string{}

	res := MatchAll(ctx, openPorts, satisfied)

	if len(res) != 1 {
		t.Fatalf("expected 1 plugin match, got %d", len(res))
	}

	if !strings.Contains(res[0].Summary, "OpenSSH") {
		t.Errorf("expected OpenSSH CVE summary, got %s", res[0].Summary)
	}

	if res[0].Port != 22 {
		t.Errorf("expected match on port 22, got %d", res[0].Port)
	}
}
