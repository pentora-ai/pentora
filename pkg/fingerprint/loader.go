// Package fingerprint provides functionality to resolve service banners into structured metadata.
package fingerprint

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed data/fingerprint_db.yaml
var embeddedFingerprintYAML []byte

// loadBuiltinRules loads fingerprint rules embedded in the binary.
func loadBuiltinRules() []StaticRule {
	rules, err := parseFingerprintYAML(embeddedFingerprintYAML)
	if err != nil {
		fmt.Printf("Failed to load embedded fingerprint rules: %v\n", err)
		return nil
	}
	return prepareRules(rules)
}

// loadExternalCatalog attempts to load fingerprint rules from workspace cache.
func loadExternalCatalog(cacheDir string) ([]StaticRule, error) {
	if cacheDir == "" {
		return nil, errors.New("cache directory not specified")
	}
	cachedPath := filepath.Join(cacheDir, "fingerprint.cache")
	content, err := os.ReadFile(cachedPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("read cache: %w", err)
	}
	rules, err := parseFingerprintYAML(content)
	if err != nil {
		return nil, fmt.Errorf("parse cache: %w", err)
	}
	return prepareRules(rules), nil
}
