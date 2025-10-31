package fingerprint

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// reset probe catalog globals for testing
func resetProbeCatalog() {
	// Force subsequent GetProbeCatalog calls to reload using embedded
	probeCatalogOnce = sync.Once{}
	probeCatalog = nil
	probeCatalogErr = nil
}

func TestGetProbeCatalog_LoadsEmbedded(t *testing.T) {
	resetProbeCatalog()
	cat, err := GetProbeCatalog()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat == nil || len(cat.Groups) == 0 {
		t.Fatalf("expected non-empty embedded catalog")
	}
}

func TestWarmProbeCatalogWithExternal_FallbackAndUse(t *testing.T) {
	dir := t.TempDir()

	// 1) No cache file -> should return nil error and keep embedded
	resetProbeCatalog()
	if err := WarmProbeCatalogWithExternal(dir); err != nil {
		t.Fatalf("unexpected error for missing cache: %v", err)
	}
	cat1, err := GetProbeCatalog()
	if err != nil || cat1 == nil {
		t.Fatalf("expected embedded catalog after missing cache, got err=%v", err)
	}

	// 2) Provide a valid external catalog and ensure it is used
	external := []byte(
		"groups:\n" +
			"  - id: custom-http\n" +
			"    description: external group\n" +
			"    port_hints: [80]\n" +
			"    protocol_hints: [http]\n" +
			"    probes:\n" +
			"      - id: p1\n" +
			"        protocol: tcp\n" +
			"        payload: \"GET / HTTP/1.0\\r\\n\\r\\n\"\n",
	)
	if err := SaveProbeCatalogCache(dir, external); err != nil {
		t.Fatalf("save cache: %v", err)
	}

	// Reset state before warming to ensure Warm can take effect
	resetProbeCatalog()
	if err := WarmProbeCatalogWithExternal(dir); err != nil {
		t.Fatalf("unexpected warm error: %v", err)
	}
	cat2, err := GetProbeCatalog()
	if err != nil {
		t.Fatalf("unexpected error getting warmed catalog: %v", err)
	}
	if cat2 == nil || len(cat2.Groups) != 1 || cat2.Groups[0].ID != "custom-http" {
		t.Fatalf("expected external catalog to be active, got: %+v", cat2)
	}

	// 3) Invalid external should return error and not change globals
	badDir := t.TempDir()
	badPath := filepath.Join(badDir, "probe.catalog.yaml")
	if err := os.WriteFile(badPath, []byte("not: yaml: ["), 0o600); err != nil {
		t.Fatalf("write bad cache: %v", err)
	}
	if err := WarmProbeCatalogWithExternal(badDir); err == nil {
		t.Fatalf("expected error when warming with invalid yaml")
	}

	// Cleanup: reset globals so other tests see embedded again
	resetProbeCatalog()
}
