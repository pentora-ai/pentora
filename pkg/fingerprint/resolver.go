package fingerprint

import "context"

// FingerprintInput defines the required fields to resolve a banner into a structured fingerprint.
type FingerprintInput struct {
	Protocol    string // Protocol type (e.g., "http", "ssh")
	Banner      string // Raw banner string retrieved from the service
	Port        int    // Port number where the service is detected
	ServiceHint string // Optional service name hint (e.g., "Pure-FTPd", "Postfix")
}

// FingerprintResult defines the normalized output of the resolution process.
type FingerprintResult struct {
	Product     string  // Product name (e.g., "LiteSpeed Web Server")
	Version     string  // Version string (e.g., "6.1")
	Vendor      string  // Vendor name (e.g., "LiteSpeed Technologies")
	CPE         string  // Normalized CPE identifier (e.g., "cpe:2.3:a:...")
	Confidence  float64 // Confidence score (0.0–1.0), especially for AI-based resolution
	Technique   string  // Technique used, e.g., "static" or "ml"
	Description string  // Optional explanation for the match
}

// FingerprintResolver is an interface that must be implemented by all resolver engines.
// This allows both rule-based and AI-based systems to be integrated seamlessly.
type FingerprintResolver interface {
	Resolve(ctx context.Context, in FingerprintInput) (FingerprintResult, error)
}
