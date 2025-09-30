// pkg/fingerprint/loader.go
// Package fingerprint provides functionality to resolve service banners into structured metadata.
package fingerprint

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
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
	return rules
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
	return rules, nil
}

// writeCatalogCache writes fingerprint rules to cache.
func writeCatalogCache(cacheDir string, data []byte) error {
	if cacheDir == "" {
		return errors.New("cache directory not specified")
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	cachedPath := filepath.Join(cacheDir, "fingerprint.cache")
	return os.WriteFile(cachedPath, data, 0o644)
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
