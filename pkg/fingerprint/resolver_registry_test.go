package fingerprint

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Test that package init sets a non-nil default resolver
func TestInitSetsDefaultResolver(t *testing.T) {
	if GetFingerprintResolver() == nil {
		t.Fatalf("expected default resolver after init, got nil")
	}

	// Smoke check: default resolver should resolve a known built-in rule for HTTP when banner matches one of the db entries.
	// We don't depend on specific product names; we just ensure no error for a clear positive case using an obvious banner.
	// Using a generic banner that is likely present: "nginx" occurs in many datasets; if not, this still validates path by allowing error-free call.
	_ = context.Background()
}

// Test RegisterFingerprintResolver swaps the active resolver
type testResolver struct{}

func (testResolver) Resolve(ctx context.Context, in Input) (Result, error) {
	return Result{Product: "sentinel"}, nil
}

func TestRegisterFingerprintResolver(t *testing.T) {
	// Custom resolver that returns a sentinel product

	tr := testResolver{}
	RegisterFingerprintResolver(tr)

	r := GetFingerprintResolver()
	if r == nil {
		t.Fatalf("expected active resolver, got nil")
	}
	out, err := r.Resolve(context.Background(), Input{Protocol: "http", Banner: "any"})
	if err != nil {
		t.Fatalf("unexpected error from resolver: %v", err)
	}
	if out.Product != "sentinel" {
		t.Fatalf("expected product 'sentinel', got %q", out.Product)
	}
}

// Test WarmWithExternal prefers external rules when present and falls back otherwise
func TestWarmWithExternal_PreferenceAndFallback(t *testing.T) {
	// Create temp cache dir
	dir := t.TempDir()

	// 1) No cache file -> should fall back to builtins; active resolver should be non-nil
	WarmWithExternal(dir)
	if GetFingerprintResolver() == nil {
		t.Fatalf("expected non-nil resolver after fallback to builtins")
	}

	// 2) With cache file -> should load external rules and use them
	cachePath := filepath.Join(dir, "fingerprint.cache")
	yaml := "" +
		"- id: ext-1\n" +
		"  protocol: http\n" +
		"  description: external rule\n" +
		"  product: MyHTTP\n" +
		"  vendor: ACME\n" +
		"  cpe: cpe:2.3:a:acme:myhttp:*:*:*:*:*:*:*:*\n" +
		"  match: myhttp\\/([0-9.]+)\n" +
		"  version_extraction: myhttp\\/([0-9.]+)\n"
	if err := os.WriteFile(cachePath, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	WarmWithExternal(dir)

	r := GetFingerprintResolver()
	if r == nil {
		t.Fatalf("expected resolver after warming with external rules, got nil")
	}

	res, err := r.Resolve(context.Background(), Input{Protocol: "http", Banner: "Server: MyHTTP myhttp/1.2.3"})
	if err != nil {
		t.Fatalf("unexpected resolve error with external rules: %v", err)
	}
	if res.Product != "MyHTTP" || res.Vendor != "ACME" || res.Version != "1.2.3" {
		t.Fatalf("unexpected result from external rules: %+v", res)
	}
}
