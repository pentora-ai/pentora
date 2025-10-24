// Command manifest-generator creates a plugin manifest from a directory of YAML plugins.
//
// Usage:
//
//	go run cmd/tools/manifest-generator/main.go -dir ./plugins -output manifest.yaml
//
// This tool scans a directory for YAML plugin files, calculates checksums,
// and generates a manifest.yaml file that can be used by the plugin downloader.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"gopkg.in/yaml.v3"
)

var (
	pluginDir  = flag.String("dir", "./plugins", "Directory containing plugin YAML files")
	outputFile = flag.String("output", "manifest.yaml", "Output manifest file path")
	baseURL    = flag.String("base-url", "https://plugins.pentora.ai/v1", "Base URL for plugin downloads")
	version    = flag.String("version", "1.0.0", "Manifest version")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Printf("Scanning plugins in: %s\n", *pluginDir)

	// Collect all plugin files
	pluginFiles, err := collectPluginFiles(*pluginDir)
	if err != nil {
		return fmt.Errorf("failed to collect plugin files: %w", err)
	}

	if len(pluginFiles) == 0 {
		return fmt.Errorf("no plugin files found in %s", *pluginDir)
	}

	fmt.Printf("Found %d plugin file(s)\n", len(pluginFiles))

	// Generate manifest entries
	var manifestEntries []plugin.PluginManifestEntry
	categoryIndex := make(map[string][]plugin.PluginDigest)

	for _, pluginPath := range pluginFiles {
		entry, digest, err := processPlugin(pluginPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to process %s: %v\n", pluginPath, err)
			continue
		}

		manifestEntries = append(manifestEntries, *entry)

		// Add to category index
		for _, category := range entry.Categories {
			categoryIndex[string(category)] = append(categoryIndex[string(category)], *digest)
		}

		fmt.Printf("  ✓ %s v%s (%s)\n", entry.Name, entry.Version, entry.Categories)
	}

	// Create manifest
	manifest := plugin.PluginManifest{
		Version: *version,
		Plugins: manifestEntries,
		Index:   categoryIndex,
	}

	// Write manifest to file
	if err := writeManifest(&manifest, *outputFile); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Printf("\n✓ Manifest generated successfully: %s\n", *outputFile)
	fmt.Printf("  Total plugins: %d\n", len(manifestEntries))
	fmt.Printf("  Categories: %d\n", len(categoryIndex))

	// Print category summary
	fmt.Println("\nCategory Summary:")
	for category, plugins := range categoryIndex {
		fmt.Printf("  %s: %d plugins\n", category, len(plugins))
	}

	return nil
}

// collectPluginFiles recursively finds all .yaml files in the directory
func collectPluginFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// processPlugin reads a plugin file, validates it, and creates a manifest entry
func processPlugin(path string) (*plugin.PluginManifestEntry, *plugin.PluginDigest, error) {
	// Read plugin file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}

	// Parse plugin with raw map to get all fields including those not in struct
	var rawPlugin map[string]interface{}
	if err := yaml.Unmarshal(data, &rawPlugin); err != nil {
		return nil, nil, fmt.Errorf("parse YAML: %w", err)
	}

	// Extract basic fields
	name, _ := rawPlugin["name"].(string)
	version, _ := rawPlugin["version"].(string)
	author, _ := rawPlugin["author"].(string)

	// Validate required fields
	if name == "" {
		return nil, nil, fmt.Errorf("missing required field: name")
	}
	if version == "" {
		return nil, nil, fmt.Errorf("missing required field: version")
	}

	// Extract metadata
	description := ""
	categoryStr := ""
	if metadata, ok := rawPlugin["metadata"].(map[string]interface{}); ok {
		if desc, ok := metadata["description"].(string); ok {
			description = desc
		}
		if cat, ok := metadata["category"].(string); ok {
			categoryStr = cat
		}
	}

	// Calculate checksum
	checksum := calculateChecksum(data)

	// Determine categories
	categories := []string{}
	if categoryStr != "" {
		categories = append(categories, categoryStr)
	}

	// Fallback: infer category from path
	if len(categories) == 0 {
		if category := inferCategoryFromPath(path); category != "" {
			categories = append(categories, string(category))
		}
	}

	// Generate download URL based on relative path
	relPath, err := filepath.Rel(*pluginDir, path)
	if err != nil {
		relPath = filepath.Base(path)
	}
	downloadURL := fmt.Sprintf("%s/%s", *baseURL, relPath)

	// Convert category strings to Category types
	var categoriesTyped []plugin.Category
	for _, catStr := range categories {
		categoriesTyped = append(categoriesTyped, plugin.Category(catStr))
	}

	// Create manifest entry
	entry := &plugin.PluginManifestEntry{
		Name:        name,
		Version:     version,
		Description: description,
		Author:      author,
		Categories:  categoriesTyped,
		URL:         downloadURL,
		Checksum:    "sha256:" + checksum,
		Size:        int64(len(data)),
	}

	// Create digest
	digest := &plugin.PluginDigest{
		Name:     name,
		Version:  version,
		Checksum: "sha256:" + checksum,
	}

	return entry, digest, nil
}

// calculateChecksum computes SHA-256 checksum of data
func calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// inferCategoryFromPath tries to determine category from file path
func inferCategoryFromPath(path string) plugin.Category {
	lowerPath := strings.ToLower(path)

	if strings.Contains(lowerPath, "/ssh/") || strings.Contains(lowerPath, "ssh-") {
		return plugin.CategorySSH
	}
	if strings.Contains(lowerPath, "/http/") || strings.Contains(lowerPath, "http-") {
		return plugin.CategoryHTTP
	}
	if strings.Contains(lowerPath, "/tls/") || strings.Contains(lowerPath, "tls-") {
		return plugin.CategoryTLS
	}
	if strings.Contains(lowerPath, "/database/") || strings.Contains(lowerPath, "db-") {
		return plugin.CategoryDatabase
	}
	if strings.Contains(lowerPath, "/network/") {
		return plugin.CategoryNetwork
	}

	return plugin.CategoryMisc
}

// writeManifest writes the manifest to a YAML file
func writeManifest(manifest *plugin.PluginManifest, outputPath string) error {
	// Create output directory if needed
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Open file for writing
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			err = cerr
		}
	}()

	// Write YAML header comment
	header := `# Pentora Plugin Manifest
# Auto-generated by manifest-generator
# DO NOT EDIT MANUALLY

`
	if _, err := io.WriteString(file, header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Encode manifest as YAML
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)

	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("encode YAML: %w", err)
	}

	return encoder.Close()
}
