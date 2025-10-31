package fingerprint

import "testing"

func TestParseFingerprintYAML_And_ValidateRules(t *testing.T) {
	// valid: two rules, one with version extraction
	yaml := []byte(
		"- id: r1\n" +
			"  protocol: http\n" +
			"  description: test\n" +
			"  product: X\n" +
			"  vendor: V\n" +
			"  cpe: cpe:2.3:a:v:x:*:*:*:*:*:*:*:*\n" +
			"  match: x\\/([0-9.]+)\n" +
			"  version_extraction: x\\/([0-9.]+)\n" +
			"- id: r2\n" +
			"  protocol: ssh\n" +
			"  description: test2\n" +
			"  product: Y\n" +
			"  vendor: W\n" +
			"  cpe: cpe:2.3:a:w:y:*:*:*:*:*:*:*:*\n" +
			"  match: y\n",
	)
	rules, err := parseFingerprintYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	// invalid: missing product and match
	bad := []byte(
		"- id: bad\n" +
			"  protocol: http\n" +
			"  vendor: z\n" +
			"  cpe: cpe:2.3:a:z:z:*:*:*:*:*:*:*:*\n",
	)
	if _, err := parseFingerprintYAML(bad); err == nil {
		t.Fatalf("expected validation error for bad rule")
	}
}
