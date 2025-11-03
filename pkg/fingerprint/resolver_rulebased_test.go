package fingerprint

import (
	"context"
	"testing"
)

func TestPrepareRules_DefaultsAndCompilation(t *testing.T) {
	rules := []StaticRule{{
		ID:                  "r1",
		Protocol:            "http",
		Match:               `server: myhttp`,
		VersionExtraction:   `myhttp/(\d+\.\d+\.\d+)`,
		ExcludePatterns:     []string{`nothttp`},
		SoftExcludePatterns: []string{`test build`},
		// PatternStrength intentionally zero to trigger defaulting
	}}

	out := prepareRules(rules)
	if len(out) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(out))
	}
	r := out[0]

	if r.PatternStrength < 0.79 || r.PatternStrength > 0.81 {
		t.Fatalf("expected default PatternStrength ~0.80, got %v", r.PatternStrength)
	}
	if r.matchRegex == nil {
		t.Fatalf("expected matchRegex compiled")
	}
	if r.versionRegex == nil {
		t.Fatalf("expected versionRegex compiled")
	}
	if len(r.excludeRegex) != 1 || len(r.softExRegex) != 1 {
		t.Fatalf("expected compiled exclude and soft-exclude regexes, got %d/%d", len(r.excludeRegex), len(r.softExRegex))
	}
}

func TestResolve_BackwardCompatibility(t *testing.T) {
	// Ensure existing behavior still works with new fields present but unused
	rules := []StaticRule{{
		ID:                "r1",
		Protocol:          "http",
		Product:           "MyHTTP",
		Vendor:            "Acme",
		CPE:               "cpe:/a:acme:myhttp",
		Match:             `server: myhttp`,
		VersionExtraction: `myhttp/(\d+\.\d+\.\d+)`,
		// New fields left empty to avoid affecting current Resolve()
	}}
	rb := NewRuleBasedResolver(rules)

	res, err := rb.Resolve(context.TODO(), Input{Protocol: "http", Banner: "Server: MyHTTP myhttp/1.2.3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Product != "MyHTTP" || res.Vendor != "Acme" || res.Version != "1.2.3" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.Confidence != 1.0 {
		t.Fatalf("expected confidence 1.0, got %v", res.Confidence)
	}
}

func TestPrepareRules_EmptyAndNoVersionRegex(t *testing.T) {
	rules := []StaticRule{{
		ID:       "r2",
		Protocol: "ssh",
		Match:    `^ssh-2.0-`,
		// no VersionExtraction
	}}
	out := prepareRules(rules)
	if len(out) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(out))
	}
	if out[0].versionRegex != nil {
		t.Fatalf("expected nil versionRegex when VersionExtraction is empty")
	}
	if out[0].matchRegex == nil {
		t.Fatalf("expected matchRegex compiled")
	}
}
