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
	if res.Confidence <= 0.5 || res.Confidence > 1.0 {
		t.Fatalf("expected confidence in (0.5,1], got %v", res.Confidence)
	}
}

func TestResolve_MultiPhaseSelectionAndPenalties(t *testing.T) {
	rules := []StaticRule{
		{
			ID:                "a",
			Protocol:          "http",
			Product:           "SvcA",
			Vendor:            "V",
			CPE:               "cpe:/a:v:svca",
			Match:             `server: svc`,
			VersionExtraction: `svc/(\d+\.\d+)`,
			PatternStrength:   0.80,
			// soft exclude matches will penalize this candidate
			SoftExcludePatterns: []string{`beta`},
		},
		{
			ID:                "b",
			Protocol:          "http",
			Product:           "SvcB",
			Vendor:            "V",
			CPE:               "cpe:/a:v:svcb",
			Match:             `server: svc`,
			VersionExtraction: `svc/(\d+\.\d+)`,
			PatternStrength:   0.75,
			// no soft excludes
			PortBonuses: []int{8080},
		},
	}
	rb := NewRuleBasedResolver(rules)

	// Banner matches both; includes "beta" to penalize A. Port 8080 gives B a small bonus.
	res, err := rb.Resolve(context.TODO(), Input{Protocol: "http", Banner: "Server: SVC svc/1.0 beta", Port: 8080})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Product != "SvcB" {
		t.Fatalf("expected SvcB to win after penalties/bonuses, got %s", res.Product)
	}
	if res.Confidence < 0.5 || res.Confidence > 1.0 {
		t.Fatalf("confidence out of expected range: %v", res.Confidence)
	}
}

func TestResolve_ThresholdFiltersLowConfidence(t *testing.T) {
	rules := []StaticRule{{
		ID:              "low",
		Protocol:        "http",
		Product:         "Low",
		Vendor:          "V",
		CPE:             "cpe:/a:v:low",
		Match:           `server: low`,
		PatternStrength: 0.40, // below threshold after defaulting
	}}
	rb := NewRuleBasedResolver(rules)
	_, err := rb.Resolve(context.TODO(), Input{Protocol: "http", Banner: "server: low"})
	if err == nil {
		t.Fatalf("expected error due to threshold filtering")
	}
}

func TestResolve_SkipsDifferentProtocolAndNonMatchingAndHardExclude(t *testing.T) {
    rules := []StaticRule{
        { // different protocol should be skipped
            ID:        "ftp",
            Protocol:  "ftp",
            Product:   "FTPd",
            Vendor:    "V",
            CPE:       "cpe:/a:v:ftpd",
            Match:     `^220`,
        },
        { // matches pattern but hard-exclude knocks it out
            ID:               "hx",
            Protocol:         "http",
            Product:          "BadSvc",
            Vendor:           "V",
            CPE:              "cpe:/a:v:bad",
            Match:            `server: bad`,
            ExcludePatterns:  []string{`blockme`},
            PatternStrength:  0.90,
        },
        { // proper candidate that should win
            ID:              "ok",
            Protocol:        "http",
            Product:         "GoodSvc",
            Vendor:          "V",
            CPE:             "cpe:/a:v:good",
            Match:           `server: good`,
            PatternStrength: 0.80,
        },
    }
    rb := NewRuleBasedResolver(rules)

    // Banner triggers: FTP rule is different protocol (skip),
    // second rule matches but contains hard-exclude token, third matches and should be selected.
    banner := "Server: GOOD\nserver: bad blockme" // includes both patterns; hard exclude applies only to bad
    res, err := rb.Resolve(context.TODO(), Input{Protocol: "http", Banner: banner})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res.Product != "GoodSvc" {
        t.Fatalf("expected GoodSvc to be selected, got %s", res.Product)
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
