// pkg/modules/evaluation/plugin_evaluation_test.go
package evaluation

import (
	"context"
	"testing"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/stretchr/testify/require"
)

func TestNewPluginEvaluationModule(t *testing.T) {
	module := NewPluginEvaluationModule()
	require.NotNil(t, module)

	meta := module.Metadata()
	require.Equal(t, pluginEvalModuleName, meta.Name)
	require.Equal(t, engine.EvaluationModuleType, meta.Type)
	require.Equal(t, pluginEvalModuleVersion, meta.Version)
}

func TestPluginEvaluationModule_Metadata(t *testing.T) {
	module := NewPluginEvaluationModule()
	meta := module.Metadata()

	// Check basic metadata
	require.NotEmpty(t, meta.ID)
	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Description)
	require.Equal(t, engine.EvaluationModuleType, meta.Type)

	// Check consumes contract
	require.NotEmpty(t, meta.Consumes, "module should consume scan results")
	consumedKeys := make([]string, len(meta.Consumes))
	for i, entry := range meta.Consumes {
		consumedKeys[i] = entry.Key
	}
	require.Contains(t, consumedKeys, "ssh.version")
	require.Contains(t, consumedKeys, "http.server")
	require.Contains(t, consumedKeys, "service.port")
	require.Contains(t, consumedKeys, "tls.version")

	// Check produces contract
	require.NotEmpty(t, meta.Produces, "module should produce vulnerabilities")
	require.Equal(t, "evaluation.vulnerabilities", meta.Produces[0].Key)
	require.Equal(t, engine.CardinalityList, meta.Produces[0].Cardinality)
}

func TestPluginEvaluationModule_Init(t *testing.T) {
	module := NewPluginEvaluationModule()

	err := module.Init("test-instance", map[string]interface{}{})
	require.NoError(t, err)

	meta := module.Metadata()
	require.Equal(t, "test-instance", meta.ID)

	// Verify embedded plugins were loaded
	require.NotNil(t, module.plugins, "plugins should be loaded")
	require.NotNil(t, module.evaluator, "evaluator should be created")

	// Count total plugins
	totalPlugins := 0
	for _, categoryPlugins := range module.plugins {
		totalPlugins += len(categoryPlugins)
	}
	require.Equal(t, 18, totalPlugins, "should load exactly 18 embedded plugins")

	// Verify plugins by category
	require.Contains(t, module.plugins, plugin.CategorySSH)
	require.Contains(t, module.plugins, plugin.CategoryHTTP)
	require.Contains(t, module.plugins, plugin.CategoryTLS)
	require.Contains(t, module.plugins, plugin.CategoryDatabase)
	require.Contains(t, module.plugins, plugin.CategoryNetwork)

	// Verify counts per category
	require.Len(t, module.plugins[plugin.CategorySSH], 4, "should have 4 SSH plugins")
	require.Len(t, module.plugins[plugin.CategoryHTTP], 4, "should have 4 HTTP plugins")
	require.Len(t, module.plugins[plugin.CategoryTLS], 4, "should have 4 TLS plugins")
	require.Len(t, module.plugins[plugin.CategoryDatabase], 3, "should have 3 Database plugins")
	require.Len(t, module.plugins[plugin.CategoryNetwork], 3, "should have 3 Network plugins")
}

func TestPluginEvaluationModule_Execute_Skeleton(t *testing.T) {
	module := NewPluginEvaluationModule()
	require.NoError(t, module.Init("test-instance", nil))

	ctx := context.Background()
	inputs := map[string]interface{}{
		"ssh.version": "7.4",
		"ssh.banner":  "OpenSSH_7.4p1",
	}

	outputChan := make(chan engine.ModuleOutput, 10)
	done := make(chan struct{})

	// Collect outputs in background
	var outputs []engine.ModuleOutput
	go func() {
		for output := range outputChan {
			outputs = append(outputs, output)
		}
		close(done)
	}()

	// Execute module
	err := module.Execute(ctx, inputs, outputChan)
	close(outputChan)
	<-done

	require.NoError(t, err)
	// Skeleton mode: no outputs yet (will be added in Step 3)
	require.Empty(t, outputs, "skeleton mode should not produce outputs")
}

func TestPluginEvaluationModuleFactory(t *testing.T) {
	module := PluginEvaluationModuleFactory()
	require.NotNil(t, module)

	meta := module.Metadata()
	require.Equal(t, pluginEvalModuleName, meta.Name)
	require.Equal(t, engine.EvaluationModuleType, meta.Type)
}

func TestPluginEvaluationModule_Registration(t *testing.T) {
	// Test that module is registered in engine registry
	module, err := engine.GetModuleInstance("test-id", pluginEvalModuleName, map[string]interface{}{})
	require.NoError(t, err)
	require.NotNil(t, module)

	meta := module.Metadata()
	require.Equal(t, pluginEvalModuleName, meta.Name)
	require.Equal(t, "test-id", meta.ID)
}

func TestVulnerabilityResult_Structure(t *testing.T) {
	// Test the vulnerability result structure
	vuln := VulnerabilityResult{
		Target:      "192.168.1.1",
		Port:        22,
		Plugin:      "ssh-weak-cipher",
		PluginType:  "evaluation",
		Severity:    "high",
		Message:     "SSH server uses weak encryption cipher",
		Remediation: "Disable CBC-mode ciphers",
		CVE:         []string{"CVE-2008-5161"},
		CWE:         []string{"CWE-326"},
		Reference:   "https://example.com/ssh-security",
		Matched:     true,
	}

	require.Equal(t, "192.168.1.1", vuln.Target)
	require.Equal(t, 22, vuln.Port)
	require.Equal(t, "high", vuln.Severity)
	require.True(t, vuln.Matched)
	require.Contains(t, vuln.CVE, "CVE-2008-5161")
}
