// Package fingerprint provides interfaces and data structures for resolving service banners
// into structured fingerprint information. It supports both rule-based and AI-based
// fingerprinting engines by defining a common interface for resolver implementations.
//
// The core types include:
//   - Input: Encapsulates the input required to resolve a service banner.
//   - Result: Represents the structured output of a fingerprinting operation, including
//     product details, version, vendor, CPE identifier, confidence score, and technique used.
//   - FingerprintResolver: An interface that must be implemented by all fingerprint resolver
//     engines, enabling seamless integration of different resolution techniques.
package fingerprint

import "context"

// Input defines the required fields to resolve a banner into a structured fingerprint.
type Input struct {
	Protocol    string // Protocol type (e.g., "http", "ssh")
	Banner      string // Raw banner string retrieved from the service
	Port        int    // Port number where the service is detected
	ServiceHint string // Optional service name hint (e.g., "Pure-FTPd", "Postfix")
}

// Result represents the result of a fingerprinting operation, containing
// detailed information about an identified product or service. It includes the product
// name, version, vendor, normalized CPE identifier, a confidence score (useful for
// AI-based or probabilistic techniques), the technique used for identification, and
// an optional description explaining the match.
type Result struct {
	Product     string  // Product name (e.g., "LiteSpeed Web Server")
	Version     string  // Version string (e.g., "6.1")
	Vendor      string  // Vendor name (e.g., "LiteSpeed Technologies")
	CPE         string  // Normalized CPE identifier (e.g., "cpe:2.3:a:...")
	Confidence  float64 // Confidence score (0.0–1.0), especially for AI-based resolution
	Technique   string  // Technique used, e.g., "static" or "ml"
	Description string  // Optional explanation for the match
}

// Resolver is an interface that must be implemented by all resolver engines.
// This allows both rule-based and AI-based systems to be integrated seamlessly.
type Resolver interface {
	Resolve(ctx context.Context, in Input) (Result, error)
}
