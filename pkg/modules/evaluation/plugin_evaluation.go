// pkg/modules/evaluation/plugin_evaluation.go
package evaluation

import (
	"context"
	"fmt"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/rs/zerolog/log"
)

const (
	pluginEvalModuleID          = "plugin-evaluation-instance"
	pluginEvalModuleName        = "plugin-evaluation"
	pluginEvalModuleDescription = "Evaluates scan results against embedded security check plugins."
	pluginEvalModuleVersion     = "0.1.0"
	pluginEvalModuleAuthor      = "Pentora Team"
)

// VulnerabilityResult represents a matched vulnerability from plugin evaluation.
type VulnerabilityResult struct {
	Target      string   `json:"target"`
	Port        int      `json:"port,omitempty"`
	Plugin      string   `json:"plugin"`
	PluginType  string   `json:"plugin_type"`
	Severity    string   `json:"severity"`
	Message     string   `json:"message"`
	Remediation string   `json:"remediation,omitempty"`
	CVE         []string `json:"cve,omitempty"`
	CWE         []string `json:"cwe,omitempty"`
	Reference   string   `json:"reference,omitempty"`
	Matched     bool     `json:"matched"`
}

// PluginEvaluationModule evaluates scan results against embedded security plugins.
type PluginEvaluationModule struct {
	meta      engine.ModuleMetadata
	plugins   map[plugin.Category][]*plugin.YAMLPlugin
	evaluator *plugin.Evaluator
}

// NewPluginEvaluationModule creates a new plugin evaluation module instance.
func NewPluginEvaluationModule() *PluginEvaluationModule {
	return &PluginEvaluationModule{
		meta: engine.ModuleMetadata{
			ID:          pluginEvalModuleID,
			Name:        pluginEvalModuleName,
			Description: pluginEvalModuleDescription,
			Version:     pluginEvalModuleVersion,
			Type:        engine.EvaluationModuleType,
			Author:      pluginEvalModuleAuthor,
			Tags:        []string{"evaluation", "plugin", "vulnerability", "security"},
			Consumes: []engine.DataContractEntry{
				{
					Key:          "ssh.version",
					DataTypeName: "string",
					Cardinality:  engine.CardinalitySingle,
					IsOptional:   true,
					Description:  "SSH version string from banner parsing",
				},
				{
					Key:          "ssh.banner",
					DataTypeName: "string",
					Cardinality:  engine.CardinalitySingle,
					IsOptional:   true,
					Description:  "Raw SSH banner string",
				},
				{
					Key:          "http.server",
					DataTypeName: "string",
					Cardinality:  engine.CardinalitySingle,
					IsOptional:   true,
					Description:  "HTTP Server header value",
				},
				{
					Key:          "service.port",
					DataTypeName: "int",
					Cardinality:  engine.CardinalitySingle,
					IsOptional:   true,
					Description:  "Service port number",
				},
				{
					Key:          "tls.version",
					DataTypeName: "string",
					Cardinality:  engine.CardinalitySingle,
					IsOptional:   true,
					Description:  "TLS protocol version",
				},
			},
			Produces: []engine.DataContractEntry{
				{
					Key:          "evaluation.vulnerabilities",
					DataTypeName: "evaluation.VulnerabilityResult",
					Cardinality:  engine.CardinalityList,
					Description:  "List of vulnerabilities detected by plugins",
				},
			},
		},
	}
}

// Metadata returns the module metadata.
func (m *PluginEvaluationModule) Metadata() engine.ModuleMetadata {
	return m.meta
}

// Init initializes the plugin evaluation module and loads embedded plugins.
func (m *PluginEvaluationModule) Init(instanceID string, config map[string]interface{}) error {
	m.meta.ID = instanceID
	logger := log.With().Str("module", m.meta.Name).Str("instance_id", m.meta.ID).Logger()

	// Load embedded plugins
	logger.Info().Msg("Loading embedded security check plugins")
	plugins, err := plugin.LoadEmbeddedPlugins()
	if err != nil {
		return fmt.Errorf("failed to load embedded plugins: %w", err)
	}

	// Store plugins in module state
	m.plugins = plugins

	// Create evaluator for plugin execution
	m.evaluator = plugin.NewEvaluator()

	// Log summary
	totalPlugins := 0
	for category, categoryPlugins := range m.plugins {
		count := len(categoryPlugins)
		totalPlugins += count
		logger.Info().
			Str("category", category.String()).
			Int("count", count).
			Msg("Loaded embedded plugins for category")
	}

	logger.Info().
		Int("total_plugins", totalPlugins).
		Msg("Plugin evaluation module initialized successfully")

	return nil
}

// Execute runs the plugin evaluation against the scan context.
func (m *PluginEvaluationModule) Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- engine.ModuleOutput) error {
	logger := log.With().Str("module", m.meta.Name).Str("instance_id", m.meta.ID).Logger()
	logger.Info().Msg("Plugin evaluation module execution started")

	// Build evaluation context from inputs
	evalContext := m.buildEvaluationContext(inputs)
	if len(evalContext) == 0 {
		logger.Warn().Msg("No evaluation context available, skipping plugin evaluation")
		return nil
	}

	logger.Info().
		Int("context_keys", len(evalContext)).
		Msg("Built evaluation context from inputs")

	// Get all plugins as flat list for evaluation
	allPlugins, err := m.getAllPluginsFlat()
	if err != nil {
		return fmt.Errorf("failed to get plugins: %w", err)
	}

	// Evaluate plugins one by one, skipping those with unsupported triggers
	matchCount := 0
	for _, pluginToEval := range allPlugins {
		result, err := m.evaluator.Evaluate(pluginToEval, evalContext)
		if err != nil {
			// Skip plugins with unsupported triggers (port, service conditions)
			logger.Debug().
				Str("plugin", pluginToEval.Name).
				Err(err).
				Msg("Skipping plugin due to evaluation error (likely unsupported trigger)")
			continue
		}

		if !result.Matched {
			continue
		}

		matchCount++

		// Extract target information from context
		target := m.extractTarget(evalContext)
		port := m.extractPort(evalContext)

		// Create vulnerability result
		vuln := VulnerabilityResult{
			Target:      target,
			Port:        port,
			Plugin:      result.Plugin.Name,
			PluginType:  string(result.Plugin.Type),
			Severity:    string(result.Output.Severity),
			Message:     result.Output.Message,
			Remediation: result.Output.Remediation,
			Reference:   result.Output.Reference,
			Matched:     true,
		}

		// Add CVE reference if available (CVE is a single string in metadata)
		if len(result.Plugin.Metadata.CVE) > 0 {
			vuln.CVE = []string{result.Plugin.Metadata.CVE}
		}

		// Send vulnerability to output channel
		outputChan <- engine.ModuleOutput{
			DataKey: "evaluation.vulnerabilities",
			Data:    vuln,
		}

		logger.Info().
			Str("plugin", result.Plugin.Name).
			Str("severity", string(result.Output.Severity)).
			Str("target", target).
			Msg("Vulnerability detected")
	}

	logger.Info().
		Int("total_plugins", len(allPlugins)).
		Int("matched_plugins", matchCount).
		Msg("Plugin evaluation completed")

	return nil
}

// buildEvaluationContext builds a map[string]any from module inputs for plugin evaluation.
func (m *PluginEvaluationModule) buildEvaluationContext(inputs map[string]interface{}) map[string]any {
	context := make(map[string]any)

	// Extract all known input keys
	knownKeys := []string{
		"ssh.version",
		"ssh.banner",
		"ssh.kex_algorithms",
		"ssh.mac_algorithms",
		"ssh.encryption_algorithms",
		"http.server",
		"http.headers",
		"service.port",
		"tls.version",
		"tls.cipher",
		"tls.cipher_suites",
		"tls.cert_expired",
		"tls.cert_self_signed",
	}

	for _, key := range knownKeys {
		if value, ok := inputs[key]; ok && value != nil {
			context[key] = value
		}
	}

	return context
}

// getAllPluginsFlat returns all plugins as a flat slice.
func (m *PluginEvaluationModule) getAllPluginsFlat() ([]*plugin.YAMLPlugin, error) {
	var allPlugins []*plugin.YAMLPlugin
	for _, categoryPlugins := range m.plugins {
		allPlugins = append(allPlugins, categoryPlugins...)
	}
	return allPlugins, nil
}

// extractTarget extracts target information from context.
func (m *PluginEvaluationModule) extractTarget(context map[string]any) string {
	// Try to get target from context (will be added in future steps)
	if target, ok := context["target"].(string); ok {
		return target
	}
	return "unknown"
}

// extractPort extracts port number from context.
func (m *PluginEvaluationModule) extractPort(context map[string]any) int {
	if port, ok := context["service.port"].(int); ok {
		return port
	}
	// Try int64 conversion (common in JSON unmarshaling)
	if port, ok := context["service.port"].(int64); ok {
		return int(port)
	}
	// Try float64 conversion (common in JSON unmarshaling)
	if port, ok := context["service.port"].(float64); ok {
		return int(port)
	}
	return 0
}

// PluginEvaluationModuleFactory is the factory function for creating plugin evaluation modules.
func PluginEvaluationModuleFactory() engine.Module {
	return NewPluginEvaluationModule()
}

func init() {
	// Register the plugin evaluation module with the engine registry
	engine.RegisterModuleFactory(pluginEvalModuleName, PluginEvaluationModuleFactory)
}
