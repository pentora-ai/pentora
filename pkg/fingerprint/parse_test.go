package fingerprint

import "testing"

func TestParseFingerprintYAML_ValidList(t *testing.T) {
	yaml := []byte(
		"- id: t1\n" +
			"  protocol: http\n" +
			"  product: Demo\n" +
			"  vendor: V\n" +
			"  cpe: cpe:/a:v:demo\n" +
			"  match: 'server: demo'\n",
	)
	rules, err := parseFingerprintYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
}

func TestParseFingerprintYAML_WrappedRules(t *testing.T) {
	yaml := []byte(
		"rules:\n" +
			"  - id: t1\n" +
			"    protocol: ssh\n" +
			"    product: OpenSSH\n" +
			"    vendor: OpenBSD\n" +
			"    cpe: cpe:/a:openbsd:openssh\n" +
			"    match: '^ssh-2.0-'\n",
	)
	rules, err := parseFingerprintYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
}

func TestParseFingerprintYAML_InvalidMissingFields(t *testing.T) {
	bad := []byte(
		"- id: bad\n" +
			"  protocol: http\n" +
			"  product: \n" +
			"  vendor: V\n" +
			"  cpe: \n" +
			"  match: ''\n",
	)
	if _, err := parseFingerprintYAML(bad); err == nil {
		t.Fatalf("expected validation error for missing required fields")
	}
}

func TestParseFingerprintYAML_Empty(t *testing.T) {
	if _, err := parseFingerprintYAML([]byte("")); err == nil {
		t.Fatalf("expected error for empty yaml")
	}
}

func TestParseFingerprintYAML_NoRulesFound(t *testing.T) {
	// valid YAML but empty rules list
	if _, err := parseFingerprintYAML([]byte("rules: []\n")); err == nil {
		t.Fatalf("expected error for no fingerprint rules found")
	}
}
