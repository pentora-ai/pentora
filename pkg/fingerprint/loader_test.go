package fingerprint

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuiltinRules_ParsesEmbeddedYAML(t *testing.T) {
	// ensure default parser is used
	parseYAMLFn = parseFingerprintYAML
	rules := loadBuiltinRules()
	if len(rules) == 0 {
		t.Fatalf("expected built-in rules to load and compile")
	}
	// spot check: compiled regexes should exist for match where provided
	for _, r := range rules {
		if r.Match != "" && r.matchRegex == nil {
			t.Fatalf("expected matchRegex compiled for rule %s", r.ID)
		}
	}
}

func TestLoadExternalCatalog_ErrorsOnMissingCache(t *testing.T) {
	_, err := loadExternalCatalog(t.TempDir())
	if err == nil {
		t.Fatalf("expected error when cache file missing")
	}
}

func TestLoadExternalCatalog_ParsesValidCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fingerprint.cache")
	yaml := []byte("- id: t1\n  protocol: http\n  product: Demo\n  vendor: V\n  cpe: cpe:/a:v:demo\n  match: 'server: demo'\n  version_extraction: 'demo/(\\d+\\.\\d+)'\n")
	if err := os.WriteFile(path, yaml, 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	rules, err := loadExternalCatalog(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].matchRegex == nil || rules[0].versionRegex == nil {
		t.Fatalf("expected compiled regexes for loaded rule")
	}
}

func TestLoadExternalCatalog_EmptyCacheDir(t *testing.T) {
	if _, err := loadExternalCatalog(""); err == nil {
		t.Fatalf("expected error when cacheDir is empty")
	}
}

func TestLoadExternalCatalog_ReadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fingerprint.cache")
	// create file but remove read permission to simulate read error
	if err := os.WriteFile(path, []byte("- id: t\n"), 0o000); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	if _, err := loadExternalCatalog(dir); err == nil {
		t.Fatalf("expected read error due to permissions")
	}
}

func TestLoadExternalCatalog_ParseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fingerprint.cache")
	// invalid YAML
	if err := os.WriteFile(path, []byte("not: [ yaml"), 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	if _, err := loadExternalCatalog(dir); err == nil {
		t.Fatalf("expected parse error for invalid yaml")
	}
}

func TestLoadBuiltinRules_ParseErrorPath(t *testing.T) {
	// stub parser to force error
	orig := parseYAMLFn
	t.Cleanup(func() { parseYAMLFn = orig })
	parseYAMLFn = func(_ []byte) ([]StaticRule, error) {
		return nil, fmt.Errorf("forced parse error")
	}
	got := loadBuiltinRules()
	if got != nil {
		t.Fatalf("expected nil when parse fails, got %v", got)
	}
}
