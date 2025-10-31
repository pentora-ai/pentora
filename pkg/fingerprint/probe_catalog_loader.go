package fingerprint

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed data/probes.yaml
var embeddedProbeCatalogYAML []byte

var (
	probeCatalogOnce sync.Once
	probeCatalog     *ProbeCatalog
	probeCatalogErr  error
)

// GetProbeCatalog returns the active probe catalog, loading the embedded definitions on first use.
func GetProbeCatalog() (*ProbeCatalog, error) {
	probeCatalogOnce.Do(func() {
		probeCatalog, probeCatalogErr = loadProbeCatalog(embeddedProbeCatalogYAML)
	})
	if probeCatalogErr != nil {
		return nil, probeCatalogErr
	}
	return probeCatalog, nil
}

// WarmProbeCatalogWithExternal attempts to load probes from a cache directory. If the cache
// is missing or invalid, it falls back to the embedded catalog.
func WarmProbeCatalogWithExternal(cacheDir string) error {
	if cacheDir == "" {
		return errors.New("cache directory not specified")
	}

	cachedPath := filepath.Join(cacheDir, "probe.catalog.yaml")
	data, err := os.ReadFile(cachedPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Nothing cached; keep embedded version.
			return nil
		}
		return fmt.Errorf("read probe catalog cache: %w", err)
	}

	catalog, err := loadProbeCatalog(data)
	if err != nil {
		return fmt.Errorf("parse probe catalog cache: %w", err)
	}

	// Replace active catalog and ensure subsequent GetProbeCatalog does not overwrite it
	probeCatalog = catalog
	probeCatalogErr = nil
	probeCatalogOnce = sync.Once{}
	// Prime the once so that future GetProbeCatalog() does not reload embedded
	probeCatalogOnce.Do(func() {})
	return nil
}

// SaveProbeCatalogCache writes the given catalog bytes to the specified cache directory.
func SaveProbeCatalogCache(cacheDir string, data []byte) error {
	if cacheDir == "" {
		return errors.New("cache directory not specified")
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	cachedPath := filepath.Join(cacheDir, "probe.catalog.yaml")
	return os.WriteFile(cachedPath, data, 0o644)
}

func loadProbeCatalog(data []byte) (*ProbeCatalog, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("probe catalog data is empty")
	}

	var catalog ProbeCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("unmarshal probe catalog: %w", err)
	}

	if err := catalog.Validate(); err != nil {
		return nil, err
	}

	return &catalog, nil
}

// ParseProbeCatalog parses raw bytes into a ProbeCatalog instance without mutating global state.
func ParseProbeCatalog(data []byte) (*ProbeCatalog, error) {
	return loadProbeCatalog(data)
}
