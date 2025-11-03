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
			ID:       "ftp",
			Protocol: "ftp",
			Product:  "FTPd",
			Vendor:   "V",
			CPE:      "cpe:/a:v:ftpd",
			Match:    `^220`,
		},
		{ // matches pattern but hard-exclude knocks it out
			ID:              "hx",
			Protocol:        "http",
			Product:         "BadSvc",
			Vendor:          "V",
			CPE:             "cpe:/a:v:bad",
			Match:           `server: bad`,
			ExcludePatterns: []string{`blockme`},
			PatternStrength: 0.90,
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

func TestResolve_RegexNonMatchingBranchIsTaken(t *testing.T) {
	rules := []StaticRule{
		{
			ID:              "nonmatch",
			Protocol:        "http",
			Product:         "Never",
			Vendor:          "V",
			CPE:             "cpe:/a:v:never",
			Match:           `^zzz`, // won't match banner
			PatternStrength: 0.90,
		},
		{
			ID:              "winner",
			Protocol:        "http",
			Product:         "Winner",
			Vendor:          "V",
			CPE:             "cpe:/a:v:winner",
			Match:           `server: ok`,
			PatternStrength: 0.80,
		},
	}
	rb := NewRuleBasedResolver(rules)
	res, err := rb.Resolve(context.TODO(), Input{Protocol: "http", Banner: "server: ok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Product != "Winner" {
		t.Fatalf("expected Winner, got %s", res.Product)
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

func TestResolve_HTTPFalsePositivePrevention(t *testing.T) {
	// Simulate two protocols with overlapping text; ensure protocol filter prevents FP
	rules := []StaticRule{
		{ID: "mysql", Protocol: "mysql", Product: "MySQL", Vendor: "Oracle", CPE: "cpe:/a:oracle:mysql", Match: `^handshake mysql`, PatternStrength: 0.9},
		{ID: "http", Protocol: "http", Product: "HTTPd", Vendor: "Generic", CPE: "cpe:/a:generic:httpd", Match: `server:`, PatternStrength: 0.8},
	}
	rb := NewRuleBasedResolver(rules)

	// Banner looks like HTTP; with Protocol=mysql resolver must not return HTTP
	if _, err := rb.Resolve(context.TODO(), Input{Protocol: "mysql", Banner: "Server: Apache"}); err == nil {
		t.Fatalf("expected no match for protocol=mysql with HTTP-like banner")
	}

	// With Protocol=http, match should succeed
	res, err := rb.Resolve(context.TODO(), Input{Protocol: "http", Banner: "server: nginx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Product != "HTTPd" {
		t.Fatalf("expected HTTPd, got %s", res.Product)
	}
}

func TestResolve_BinaryVerificationHints(t *testing.T) {
	// Use BinaryMinLength and BinaryMagic as hints; current resolver doesn't hard-check
	// but ensure rules compile and still match text banners correctly.
	rules := []StaticRule{
		{ID: "redis", Protocol: "redis", Product: "Redis", Vendor: "Redis", CPE: "cpe:/a:redislabs:redis", Match: `^\+ok|^-err|^\$`, PatternStrength: 0.85, BinaryMinLength: 0, BinaryMagic: []string{"\x2a\x31"}},
	}
	rb := NewRuleBasedResolver(rules)
	// A simple Redis-like banner
	_, err := rb.Resolve(context.TODO(), Input{Protocol: "redis", Banner: "+ok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolve_PrefersHigherConfidenceWithBonusesAndPenalties(t *testing.T) {
	rules := []StaticRule{
		{
			ID:                  "high-base-with-penalty",
			Protocol:            "http",
			Product:             "SvcHigh",
			Vendor:              "V",
			CPE:                 "cpe:/a:v:svchigh",
			Match:               `server: svcx`,
			VersionExtraction:   `svcx/(\d+\.\d+)`,
			PatternStrength:     0.90,
			SoftExcludePatterns: []string{`beta`}, // will penalize
		},
		{
			ID:                "lower-base-with-bonus",
			Protocol:          "http",
			Product:           "SvcLow",
			Vendor:            "V",
			CPE:               "cpe:/a:v:svclow",
			Match:             `server: svcx`,
			VersionExtraction: `svcx/(\d+\.\d+)`,
			PatternStrength:   0.80,
			PortBonuses:       []int{8080},
		},
	}
	rb := NewRuleBasedResolver(rules)
	// Banner matches both; contains "beta" penalizing SvcHigh; port bonus helps SvcLow.
	res, err := rb.Resolve(context.TODO(), Input{Protocol: "http", Banner: "Server: SVCX svcx/2.1 beta", Port: 8080})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Product != "SvcLow" {
		t.Fatalf("expected SvcLow to win after penalties/bonuses, got %s", res.Product)
	}
	if res.Version != "2.1" {
		t.Fatalf("expected version 2.1, got %s", res.Version)
	}
}

func TestResolve_HardExcludeBeatsPatternMatch(t *testing.T) {
	rules := []StaticRule{
		{
			ID:              "bad",
			Protocol:        "http",
			Product:         "Bad",
			Vendor:          "V",
			CPE:             "cpe:/a:v:bad",
			Match:           `server: bad`,
			ExcludePatterns: []string{`block`},
			PatternStrength: 0.95,
		},
		{
			ID:              "good",
			Protocol:        "http",
			Product:         "Good",
			Vendor:          "V",
			CPE:             "cpe:/a:v:good",
			Match:           `server: good`,
			PatternStrength: 0.60,
		},
	}
	rb := NewRuleBasedResolver(rules)
	// Contains both patterns; hard exclude should eliminate "bad" despite higher base strength.
	banner := "server: bad block\nserver: good"
	res, err := rb.Resolve(context.TODO(), Input{Protocol: "http", Banner: banner})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Product != "Good" {
		t.Fatalf("expected Good due to hard-exclude on Bad, got %s", res.Product)
	}
}

func TestResolve_ReturnsErrorWhenNoCandidatesPassThreshold(t *testing.T) {
	rules := []StaticRule{{
		ID:              "weak",
		Protocol:        "http",
		Product:         "Weak",
		Vendor:          "V",
		CPE:             "cpe:/a:v:weak",
		Match:           `server: weak`,
		PatternStrength: 0.45, // will remain below 0.5 after scoring
	}}
	rb := NewRuleBasedResolver(rules)
	if _, err := rb.Resolve(context.TODO(), Input{Protocol: "http", Banner: "server: weak"}); err == nil {
		t.Fatalf("expected error due to threshold filtering of all candidates")
	}
}

// MySQL Perfect Implementation Tests (Phase 3)

func TestResolve_MySQLHandshake_MySQL57(t *testing.T) {
	rules := []StaticRule{{
		ID:                  "mysql.mysql",
		Protocol:            "mysql",
		Product:             "MySQL",
		Vendor:              "Oracle",
		CPE:                 "cpe:2.3:a:oracle:mysql:*:*:*:*:*:*:*:*",
		Match:               `\x00\x00\x00\x0a`,
		VersionExtraction:   `\x0a([\d\.p]+[\w\-]*)`,
		ExcludePatterns:     []string{`http/`, `<html`, `<!doctype`, `<body`},
		SoftExcludePatterns: []string{`error`, `denied`, `refused`, `unavailable`},
		PatternStrength:     0.90,
		PortBonuses:         []int{3306, 33060},
		BinaryMinLength:     10,
		BinaryMagic:         []string{`\x00\x00\x00\x0a`},
	}}
	rb := NewRuleBasedResolver(rules)

	// Simulate MySQL 5.7 handshake banner
	banner := "\x00\x00\x00\x0a5.7.44-log\x00"
	res, err := rb.Resolve(context.TODO(), Input{Protocol: "mysql", Banner: banner, Port: 3306})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Product != "MySQL" {
		t.Fatalf("expected MySQL, got %s", res.Product)
	}
	if res.Version != "5.7.44-log" {
		t.Fatalf("expected version 5.7.44-log, got %s", res.Version)
	}
	if res.Confidence < 0.90 {
		t.Fatalf("expected high confidence (>0.90), got %v", res.Confidence)
	}
}

func TestResolve_MySQLHandshake_MySQL80(t *testing.T) {
	rules := []StaticRule{{
		ID:                  "mysql.mysql",
		Protocol:            "mysql",
		Product:             "MySQL",
		Vendor:              "Oracle",
		CPE:                 "cpe:2.3:a:oracle:mysql:*:*:*:*:*:*:*:*",
		Match:               `\x00\x00\x00\x0a`,
		VersionExtraction:   `\x0a([\d\.p]+[\w\-]*)`,
		ExcludePatterns:     []string{`http/`, `<html`, `<!doctype`, `<body`},
		SoftExcludePatterns: []string{`error`, `denied`, `refused`, `unavailable`},
		PatternStrength:     0.90,
		PortBonuses:         []int{3306, 33060},
		BinaryMinLength:     10,
		BinaryMagic:         []string{`\x00\x00\x00\x0a`},
	}}
	rb := NewRuleBasedResolver(rules)

	// Simulate MySQL 8.0 handshake banner
	banner := "\x00\x00\x00\x0a8.0.35\x00"
	res, err := rb.Resolve(context.TODO(), Input{Protocol: "mysql", Banner: banner, Port: 3306})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Product != "MySQL" {
		t.Fatalf("expected MySQL, got %s", res.Product)
	}
	if res.Version != "8.0.35" {
		t.Fatalf("expected version 8.0.35, got %s", res.Version)
	}
	if res.Confidence < 0.90 {
		t.Fatalf("expected high confidence (>0.90), got %v", res.Confidence)
	}
}

func TestResolve_MySQLHandshake_MariaDB(t *testing.T) {
	rules := []StaticRule{{
		ID:                  "mysql.mysql",
		Protocol:            "mysql",
		Product:             "MySQL",
		Vendor:              "Oracle",
		CPE:                 "cpe:2.3:a:oracle:mysql:*:*:*:*:*:*:*:*",
		Match:               `\x00\x00\x00\x0a`,
		VersionExtraction:   `\x0a([\d\.p]+[\w\-]*)`,
		ExcludePatterns:     []string{`http/`, `<html`, `<!doctype`, `<body`},
		SoftExcludePatterns: []string{`error`, `denied`, `refused`, `unavailable`},
		PatternStrength:     0.90,
		PortBonuses:         []int{3306, 33060},
		BinaryMinLength:     10,
		BinaryMagic:         []string{`\x00\x00\x00\x0a`},
	}}
	rb := NewRuleBasedResolver(rules)

	// Simulate MariaDB handshake banner
	banner := "\x00\x00\x00\x0a10.11.6-MariaDB\x00"
	res, err := rb.Resolve(context.TODO(), Input{Protocol: "mysql", Banner: banner, Port: 3306})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Product != "MySQL" {
		t.Fatalf("expected MySQL (MariaDB compatible), got %s", res.Product)
	}
	if res.Version != "10.11.6-mariadb" {
		t.Fatalf("expected version 10.11.6-mariadb, got %s", res.Version)
	}
	if res.Confidence < 0.90 {
		t.Fatalf("expected high confidence (>0.90), got %v", res.Confidence)
	}
}

func TestResolve_MySQLHandshake_PortBonus(t *testing.T) {
	rules := []StaticRule{{
		ID:                  "mysql.mysql",
		Protocol:            "mysql",
		Product:             "MySQL",
		Vendor:              "Oracle",
		CPE:                 "cpe:2.3:a:oracle:mysql:*:*:*:*:*:*:*:*",
		Match:               `\x00\x00\x00\x0a`,
		VersionExtraction:   `\x0a([\d\.p]+[\w\-]*)`,
		ExcludePatterns:     []string{`http/`, `<html`, `<!doctype`, `<body`},
		SoftExcludePatterns: []string{`error`, `denied`, `refused`, `unavailable`},
		PatternStrength:     0.90,
		PortBonuses:         []int{3306, 33060},
		BinaryMinLength:     10,
		BinaryMagic:         []string{`\x00\x00\x00\x0a`},
	}}
	rb := NewRuleBasedResolver(rules)

	banner := "\x00\x00\x00\x0a8.0.35\x00"

	// Test with standard MySQL port (should get bonus)
	resWithBonus, err := rb.Resolve(context.TODO(), Input{Protocol: "mysql", Banner: banner, Port: 3306})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test with non-standard port (no bonus)
	resNoBonus, err := rb.Resolve(context.TODO(), Input{Protocol: "mysql", Banner: banner, Port: 9999})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Confidence should be higher with port bonus
	if resWithBonus.Confidence <= resNoBonus.Confidence {
		t.Fatalf("expected port bonus to increase confidence: with=%v, without=%v", resWithBonus.Confidence, resNoBonus.Confidence)
	}
}

func TestResolve_MySQLHandshake_HTTPFalsePositiveRejection(t *testing.T) {
	rules := []StaticRule{{
		ID:                  "mysql.mysql",
		Protocol:            "mysql",
		Product:             "MySQL",
		Vendor:              "Oracle",
		CPE:                 "cpe:2.3:a:oracle:mysql:*:*:*:*:*:*:*:*",
		Match:               `\x00\x00\x00\x0a`,
		VersionExtraction:   `\x0a([\d\.p]+[\w\-]*)`,
		ExcludePatterns:     []string{`http/`, `<html`, `<!doctype`, `<body`},
		SoftExcludePatterns: []string{`error`, `denied`, `refused`, `unavailable`},
		PatternStrength:     0.90,
		PortBonuses:         []int{3306, 33060},
		BinaryMinLength:     10,
		BinaryMagic:         []string{`\x00\x00\x00\x0a`},
	}}
	rb := NewRuleBasedResolver(rules)

	testCases := []struct {
		name   string
		banner string
	}{
		{"HTTP Response Header", "http/1.1 200 ok\r\ncontent-type: text/html\r\n\r\n\x00\x00\x00\x0amysql"},
		{"HTML Document", "<html><body>\x00\x00\x00\x0amysql</body></html>"},
		{"HTML Doctype", "<!doctype html>\x00\x00\x00\x0amysql"},
		{"HTML Body Tag", "<body>\x00\x00\x00\x0amysql</body>"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := rb.Resolve(context.TODO(), Input{Protocol: "mysql", Banner: tc.banner})
			if err == nil {
				t.Fatalf("expected HTTP false positive to be rejected for: %s", tc.name)
			}
		})
	}
}

func TestResolve_MySQLHandshake_SoftExcludePenalty(t *testing.T) {
	rules := []StaticRule{{
		ID:                  "mysql.mysql",
		Protocol:            "mysql",
		Product:             "MySQL",
		Vendor:              "Oracle",
		CPE:                 "cpe:2.3:a:oracle:mysql:*:*:*:*:*:*:*:*",
		Match:               `\x00\x00\x00\x0a`,
		VersionExtraction:   `\x0a([\d\.p]+[\w\-]*)`,
		ExcludePatterns:     []string{`http/`, `<html`, `<!doctype`, `<body`},
		SoftExcludePatterns: []string{`error`, `denied`, `refused`, `unavailable`},
		PatternStrength:     0.90,
		PortBonuses:         []int{3306, 33060},
		BinaryMinLength:     10,
		BinaryMagic:         []string{`\x00\x00\x00\x0a`},
	}}
	rb := NewRuleBasedResolver(rules)

	// Normal MySQL handshake
	normalBanner := "\x00\x00\x00\x0a8.0.35\x00"
	normalRes, err := rb.Resolve(context.TODO(), Input{Protocol: "mysql", Banner: normalBanner, Port: 3306})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// MySQL handshake with error text (should get penalized)
	errorBanner := "\x00\x00\x00\x0a8.0.35\x00 access denied error"
	errorRes, err := rb.Resolve(context.TODO(), Input{Protocol: "mysql", Banner: errorBanner, Port: 3306})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Error banner should have lower confidence due to soft exclude penalty
	if errorRes.Confidence >= normalRes.Confidence {
		t.Fatalf("expected soft exclude to penalize confidence: normal=%v, error=%v", normalRes.Confidence, errorRes.Confidence)
	}
}

func TestResolve_MySQLHandshake_VersionExtractionEdgeCases(t *testing.T) {
	rules := []StaticRule{{
		ID:                  "mysql.mysql",
		Protocol:            "mysql",
		Product:             "MySQL",
		Vendor:              "Oracle",
		CPE:                 "cpe:2.3:a:oracle:mysql:*:*:*:*:*:*:*:*",
		Match:               `\x00\x00\x00\x0a`,
		VersionExtraction:   `\x0a([\d\.p]+[\w\-]*)`,
		ExcludePatterns:     []string{`http/`, `<html`, `<!doctype`, `<body`},
		SoftExcludePatterns: []string{`error`, `denied`, `refused`, `unavailable`},
		PatternStrength:     0.90,
		PortBonuses:         []int{3306, 33060},
		BinaryMinLength:     10,
		BinaryMagic:         []string{`\x00\x00\x00\x0a`},
	}}
	rb := NewRuleBasedResolver(rules)

	testCases := []struct {
		name            string
		banner          string
		expectedVersion string
	}{
		{"Simple Version", "\x00\x00\x00\x0a8.0.35\x00", "8.0.35"},
		{"Version with Patch", "\x00\x00\x00\x0a5.7.44p1\x00", "5.7.44p1"},
		{"Version with Suffix", "\x00\x00\x00\x0a8.0.35-log\x00", "8.0.35-log"},
		{"MariaDB Version", "\x00\x00\x00\x0a10.11.6-MariaDB\x00", "10.11.6-mariadb"},
		{"Percona Version", "\x00\x00\x00\x0a8.0.35-27-percona\x00", "8.0.35-27-percona"},
		{"No Version", "\x00\x00\x00\x0a\x00", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := rb.Resolve(context.TODO(), Input{Protocol: "mysql", Banner: tc.banner, Port: 3306})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Version != tc.expectedVersion {
				t.Fatalf("expected version %s, got %s", tc.expectedVersion, res.Version)
			}
		})
	}
}
