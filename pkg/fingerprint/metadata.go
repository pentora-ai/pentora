package fingerprint

// Catalog represents a set of fingerprint definitions that can be shipped with the binary
// or fetched externally. The structure is intentionally flexible so future signals
// (TLS SNI, JA3, HTTP headers, etc.) can be added without breaking compatibility.
type Catalog struct {
	Source       string                 `yaml:"source" json:"source"` // e.g. builtin, community, enterprise
	Version      string                 `yaml:"version" json:"version"`
	Fingerprints []Metadata             `yaml:"fingerprints" json:"fingerprints"`
	Metadata     map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// Metadata describes how to recognise a specific service/product.
type Metadata struct {
	ID          string            `yaml:"id" json:"id"`
	Protocol    string            `yaml:"protocol" json:"protocol"` // ssh, http, rdp, smb ...
	Ports       []int             `yaml:"ports,omitempty" json:"ports,omitempty"`
	Product     string            `yaml:"product,omitempty" json:"product,omitempty"`
	Vendor      string            `yaml:"vendor,omitempty" json:"vendor,omitempty"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	CPEs        []string          `yaml:"cpes,omitempty" json:"cpes,omitempty"`
	Matchers    []Matcher         `yaml:"matchers" json:"matchers"`
	Probes      []ProbeDefinition `yaml:"probes,omitempty" json:"probes,omitempty"`
	Priority    int               `yaml:"priority,omitempty" json:"priority,omitempty"`
	Tags        []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Matcher defines how to evaluate an incoming signal (banner, TLS info, etc.).
type Matcher struct {
	Type      MatcherType `yaml:"type" json:"type"`
	Pattern   string      `yaml:"pattern" json:"pattern"` // regex/glob/contains depends on type
	Transform string      `yaml:"transform,omitempty" json:"transform,omitempty"`
}

// MatcherType enumerates supported matcher styles.
type MatcherType string

// MatcherContains specifies a matcher type that checks if a value contains a specified substring.
const (
	MatcherContains MatcherType = "contains" // Case-insensitive substring match
	MatcherPrefix   MatcherType = "prefix"   // Case-insensitive prefix match
	MatcherRegex    MatcherType = "regex"    // Case-insensitive regex match
	MatcherEquals   MatcherType = "equals"   // Case-insensitive exact match
)

// ProbeDefinition describes an active probe to run when matching a service.
type ProbeDefinition struct {
	ID          string            `yaml:"id" json:"id"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Payload     string            `yaml:"payload" json:"payload"`
	Protocol    string            `yaml:"protocol,omitempty" json:"protocol,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Timeout     string            `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}
