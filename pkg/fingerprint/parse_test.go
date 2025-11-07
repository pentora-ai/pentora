package fingerprint

import (
	"testing"
)

func TestParseFingerprintYAML_List(t *testing.T) {
	yml := []byte(`
 - id: t1
   protocol: http
   product: Demo
   vendor: V
   cpe: cpe:/a:v:demo
   match: 'server: demo'
`)
	rules, err := parseFingerprintYAML(yml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Protocol != "http" || rules[0].Product != "Demo" {
		t.Fatalf("parsed rule mismatch: %+v", rules[0])
	}
}

func TestParseFingerprintYAML_Wrapper(t *testing.T) {
	yml := []byte(`
 rules:
   - id: t2
     protocol: ssh
     product: OpenSSH
     vendor: OpenBSD
     cpe: cpe:/a:openbsd:openssh
     match: 'openssh'
`)
	rules, err := parseFingerprintYAML(yml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Protocol != "ssh" || rules[0].Product != "OpenSSH" {
		t.Fatalf("parsed rule mismatch: %+v", rules[0])
	}
}

func TestParseFingerprintYAML_Invalid(t *testing.T) {
	yml := []byte("not: [ yaml")
	if _, err := parseFingerprintYAML(yml); err == nil {
		t.Fatalf("expected parse error for invalid yaml")
	}
}
