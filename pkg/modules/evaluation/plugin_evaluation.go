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

	// TODO: Load embedded plugins (Step 2)
	// TODO: Build evaluation context from inputs (Step 3)
	// TODO: Evaluate plugins against context (Step 3)
	// TODO: Send vulnerability results to output channel (Step 3)

	logger.Info().Msg("Plugin evaluation completed (skeleton mode)")
	return nil
}

// PluginEvaluationModuleFactory is the factory function for creating plugin evaluation modules.
func PluginEvaluationModuleFactory() engine.Module {
	return NewPluginEvaluationModule()
}

func init() {
	// Register the plugin evaluation module with the engine registry
	engine.RegisterModuleFactory(pluginEvalModuleName, PluginEvaluationModuleFactory)
}
