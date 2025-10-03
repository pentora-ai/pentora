package fingerprint

import (
	"fmt"
	"strings"
)

// ProbeCatalog represents the set of active probe groups that the scanner can execute.
// Groups can map to protocol families (HTTP, SMTP, etc.) and expose one or more probe
// definitions that should be attempted when their match conditions are satisfied.
type ProbeCatalog struct {
	Groups []ProbeGroup `yaml:"groups" json:"groups"`
}

// ProbeGroup describes a family of probes sharing similar trigger conditions.
type ProbeGroup struct {
	ID             string      `yaml:"id" json:"id"`
	Description    string      `yaml:"description,omitempty" json:"description,omitempty"`
	PortHints      []int       `yaml:"port_hints,omitempty" json:"port_hints,omitempty"`
	ProtocolHints  []string    `yaml:"protocol_hints,omitempty" json:"protocol_hints,omitempty"`
	Probes         []ProbeSpec `yaml:"probes" json:"probes"`
	AdditionalTags []string    `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// ProbeSpec describes a single active probe that banner_grab can execute.
// The payload is a raw string that will be written as-is to the remote endpoint.
type ProbeSpec struct {
	ID              string            `yaml:"id" json:"id"`
	Description     string            `yaml:"description,omitempty" json:"description,omitempty"`
	Protocol        string            `yaml:"protocol" json:"protocol"`
	UseTLS          bool              `yaml:"use_tls,omitempty" json:"use_tls,omitempty"`
	Payload         string            `yaml:"payload" json:"payload"`
	Headers         map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Timeout         string            `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	PortInclude     []int             `yaml:"port_include,omitempty" json:"port_include,omitempty"`
	PortExclude     []int             `yaml:"port_exclude,omitempty" json:"port_exclude,omitempty"`
	SkipInitialRead bool              `yaml:"skip_initial_read,omitempty" json:"skip_initial_read,omitempty"`
}

// ProbesFor filters catalog probes for the given port and optional protocol hints.
// It returns every probe whose group matches the port/hints and whose own include/exclude
// clauses allow the port.
func (c *ProbeCatalog) ProbesFor(port int, hints []string) []ProbeSpec {
	if c == nil {
		return nil
	}

	normalizedHints := normalizeHints(hints)
	out := make([]ProbeSpec, 0)

	for _, group := range c.Groups {
		if !group.matches(port, normalizedHints) {
			continue
		}

		for _, probe := range group.Probes {
			if !probe.allowsPort(port) {
				continue
			}
			out = append(out, probe)
		}
	}

	return out
}

func (g ProbeGroup) matches(port int, hints map[string]struct{}) bool {
	if len(g.PortHints) > 0 && !containsInt(g.PortHints, port) {
		return false
	}

	if len(g.ProtocolHints) == 0 {
		return true
	}

	if hints == nil {
		// Port matched but we have no protocol hints; still allow match so long as port hint existed.
		if len(g.PortHints) > 0 {
			return true
		}
		return false
	}

	for _, hint := range g.ProtocolHints {
		if _, ok := hints[strings.ToLower(hint)]; ok {
			return true
		}
	}
	return false
}

func (p ProbeSpec) allowsPort(port int) bool {
	if len(p.PortInclude) > 0 && !containsInt(p.PortInclude, port) {
		return false
	}
	if len(p.PortExclude) > 0 && containsInt(p.PortExclude, port) {
		return false
	}
	return true
}

func containsInt(list []int, target int) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func normalizeHints(hints []string) map[string]struct{} {
	if len(hints) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(hints))
	for _, hint := range hints {
		if hint == "" {
			continue
		}
		out[strings.ToLower(hint)] = struct{}{}
	}
	return out
}

// Validate ensures catalog content is well-formed.
func (c *ProbeCatalog) Validate() error {
	if c == nil {
		return fmt.Errorf("catalog is nil")
	}
	for i, group := range c.Groups {
		if group.ID == "" {
			return fmt.Errorf("probe group at index %d is missing id", i)
		}
		if len(group.Probes) == 0 {
			return fmt.Errorf("probe group %q has no probes", group.ID)
		}
		for j, probe := range group.Probes {
			if probe.ID == "" {
				return fmt.Errorf("group %q probe at index %d is missing id", group.ID, j)
			}
			if probe.Protocol == "" {
				return fmt.Errorf("probe %q missing protocol", probe.ID)
			}
			if probe.Payload == "" {
				return fmt.Errorf("probe %q missing payload", probe.ID)
			}
		}
	}
	return nil
}
