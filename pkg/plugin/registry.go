package plugin

// Plugin represents a pluggable vulnerability rule
type Plugins struct {
	ID           string
	Name         string
	RequirePorts []int
	RequireKeys  []string
	DependsOn    []string
	MatchFunc    func(ctx map[string]string) *MatchResult
}

// MatchResult represents the output of a plugin if it detects something
type MatchResult struct {
	CVE     []string
	Summary string
	Port    int
	Info    string
}

var registry = map[string]*Plugins{}

// Register adds a plugin to the global registry
func Register(p *Plugins) {
	registry[p.ID] = p
}

// Filter returns plugins that match the given context
func Filter(ctx map[string]string, openPorts []int, satisfied []string) []*Plugins {
	var selected []*Plugins

	portMap := make(map[int]bool)
	for _, p := range openPorts {
		portMap[p] = true
	}
	satisfiedMap := make(map[string]bool)
	for _, s := range satisfied {
		satisfiedMap[s] = true
	}

	for _, p := range registry {
		match := true
		for _, reqPort := range p.RequirePorts {
			if !portMap[reqPort] {
				match = false
				break
			}
		}
		for _, reqKey := range p.RequireKeys {
			if _, ok := ctx[reqKey]; !ok {
				match = false
				break
			}
		}
		for _, dep := range p.DependsOn {
			if !satisfiedMap[dep] {
				match = false
				break
			}
		}
		if match {
			selected = append(selected, p)
		}
	}
	return selected
}

// MatchAll runs all matching plugins and returns their results
func MatchAll(ctx map[string]string, openPorts []int, satisfied []string) []*MatchResult {
	var results []*MatchResult
	for _, plugin := range Filter(ctx, openPorts, satisfied) {
		if res := plugin.MatchFunc(ctx); res != nil {
			results = append(results, res)
		}
	}
	return results
}

// Run executes the plugin's match function and returns the result
