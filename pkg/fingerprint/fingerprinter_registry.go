package fingerprint

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	registryMu     sync.RWMutex
	fingerprinters = make(map[string]Fingerprinter) // ID → Fingerprinter
	protocolIndex  = make(map[string][]string)      // protocol → []ID (sorted by priority)

	// Core protocols that are fundamental to the system.
	// Extended implementations cannot shadow these protocols.
	coreProtocols = map[string]bool{
		"ssh":   true,
		"http":  true,
		"https": true,
		"smtp":  true,
		"ftp":   true,
		"dns":   true,
	}
)

// Namespace constants for fingerprinter ID prefixes.
const (
	NamespaceBuiltin  = "builtin"
	NamespaceExtended = "extended"
	NamespaceCustom   = "custom"
	NamespacePlugin   = "plugin"
)

// RegisterFingerprinter adds a fingerprinter to the global registry with validation.
//
// The fingerprinter ID must use one of the valid namespace prefixes:
//   - builtin.*   - Core built-in fingerprinters
//   - extended.*  - Advanced fingerprinters with enhanced features
//   - custom.*    - User-defined fingerprinters
//   - plugin.*    - Plugin-provided fingerprinters
//
// Panics if:
//   - ID has an invalid namespace prefix
//   - A fingerprinter with the same ID is already registered (collision)
//   - An extended fingerprinter attempts to shadow a core protocol
func RegisterFingerprinter(fp Fingerprinter) {
	registryMu.Lock()
	defer registryMu.Unlock()

	id := fp.ID()

	// 1. Namespace validation
	if err := validateNamespace(id); err != nil {
		panic(fmt.Sprintf("invalid fingerprinter ID %q: %v", id, err)) //nolint:forbidigo // Fail-fast during init() is correct for registry APIs
	}

	// 2. Collision detection
	if _, exists := fingerprinters[id]; exists {
		panic(fmt.Sprintf("duplicate fingerprinter ID: %s (fingerprinter already registered)", id)) //nolint:forbidigo // Fail-fast during init() is correct for registry APIs
	}

	// 3. Core protocol protection
	protocol := extractProtocol(id)
	if isExtendedOverridingCore(id, protocol) {
		panic(fmt.Sprintf("cannot override core protocol: %s (extended.%s not allowed, use custom.%s or plugin.%s instead)", //nolint:forbidigo // Fail-fast during init() is correct for registry APIs
			protocol, protocol, protocol, protocol))
	}

	// Register fingerprinter
	fingerprinters[id] = fp

	// Update protocol index
	if protocol != "" {
		protocolIndex[protocol] = append(protocolIndex[protocol], id)
		sortProtocolByPriority(protocol)
	}
}

// validateNamespace checks if the fingerprinter ID has a valid namespace prefix.
func validateNamespace(id string) error {
	validPrefixes := []string{
		NamespaceBuiltin + ".",
		NamespaceExtended + ".",
		NamespaceCustom + ".",
		NamespacePlugin + ".",
	}

	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(id, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return fmt.Errorf("ID must start with one of: builtin., extended., custom., plugin. (got: %s)", id)
	}

	// Ensure protocol part is not empty
	protocol := extractProtocol(id)
	if protocol == "" {
		return fmt.Errorf("ID must have a protocol after namespace (got: %s)", id)
	}

	return nil
}

// extractProtocol extracts the protocol name from a fingerprinter ID.
// Example: "builtin.ssh" → "ssh"
func extractProtocol(id string) string {
	parts := strings.SplitN(id, ".", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// isExtendedOverridingCore checks if an extended fingerprinter tries to override a core protocol.
func isExtendedOverridingCore(id, protocol string) bool {
	if !coreProtocols[protocol] {
		return false
	}

	// Extended implementations cannot shadow core protocols
	if strings.HasPrefix(id, NamespaceExtended+".") {
		// Check if builtin version exists
		builtinID := NamespaceBuiltin + "." + protocol
		if _, exists := fingerprinters[builtinID]; exists {
			return true
		}
	}

	return false
}

// sortProtocolByPriority sorts fingerprinter IDs for a protocol by priority (highest first).
func sortProtocolByPriority(protocol string) {
	ids := protocolIndex[protocol]
	sort.SliceStable(ids, func(i, j int) bool {
		fpI := fingerprinters[ids[i]]
		fpJ := fingerprinters[ids[j]]
		return fpI.Priority() > fpJ.Priority()
	})
	protocolIndex[protocol] = ids
}

// GetFingerprinter returns the highest priority fingerprinter for a protocol.
// Returns nil if no fingerprinter is registered for the protocol.
func GetFingerprinter(protocol string) Fingerprinter {
	registryMu.RLock()
	defer registryMu.RUnlock()

	ids, ok := protocolIndex[protocol]
	if !ok || len(ids) == 0 {
		return nil
	}

	// Return highest priority (first in sorted list)
	return fingerprinters[ids[0]]
}

// GetFingerprintersByProtocol returns all fingerprinters for a protocol, sorted by priority (highest first).
func GetFingerprintersByProtocol(protocol string) []Fingerprinter {
	registryMu.RLock()
	defer registryMu.RUnlock()

	ids, ok := protocolIndex[protocol]
	if !ok {
		return nil
	}

	result := make([]Fingerprinter, 0, len(ids))
	for _, id := range ids {
		result = append(result, fingerprinters[id])
	}
	return result
}

// ListFingerprinters returns a snapshot of all registered fingerprinters.
func ListFingerprinters() []Fingerprinter {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make([]Fingerprinter, 0, len(fingerprinters))
	for _, fp := range fingerprinters {
		result = append(result, fp)
	}
	return result
}

// RegistryStats contains statistics about registered fingerprinters.
type RegistryStats struct {
	Total          int
	ByNamespace    map[string]int
	ByProtocol     map[string]int
	Fingerprinters []FingerprinterMeta
}

// FingerprinterMeta contains metadata about a registered fingerprinter.
type FingerprinterMeta struct {
	ID        string   `json:"id"`
	Namespace string   `json:"namespace"`
	Protocol  string   `json:"protocol"`
	Priority  Priority `json:"priority"`
}

// GetRegistryStats returns statistics about registered fingerprinters.
func GetRegistryStats() RegistryStats {
	registryMu.RLock()
	defer registryMu.RUnlock()

	stats := RegistryStats{
		Total:          len(fingerprinters),
		ByNamespace:    make(map[string]int),
		ByProtocol:     make(map[string]int),
		Fingerprinters: make([]FingerprinterMeta, 0, len(fingerprinters)),
	}

	for id, fp := range fingerprinters {
		namespace := strings.SplitN(id, ".", 2)[0]
		protocol := extractProtocol(id)

		stats.ByNamespace[namespace]++
		if protocol != "" {
			stats.ByProtocol[protocol]++
		}

		stats.Fingerprinters = append(stats.Fingerprinters, FingerprinterMeta{
			ID:        id,
			Namespace: namespace,
			Protocol:  protocol,
			Priority:  fp.Priority(),
		})
	}

	// Sort by priority (highest first) then by ID for deterministic order
	sort.Slice(stats.Fingerprinters, func(i, j int) bool {
		if stats.Fingerprinters[i].Priority != stats.Fingerprinters[j].Priority {
			return stats.Fingerprinters[i].Priority > stats.Fingerprinters[j].Priority
		}
		return stats.Fingerprinters[i].ID < stats.Fingerprinters[j].ID
	})

	return stats
}

// NewDefaultCoordinator returns a Coordinator pre-populated with all registered fingerprinters.
func NewDefaultCoordinator() *Coordinator {
	registryMu.RLock()
	defer registryMu.RUnlock()

	fps := make([]Fingerprinter, 0, len(fingerprinters))
	for _, fp := range fingerprinters {
		fps = append(fps, fp)
	}
	return NewCoordinator(fps...)
}
