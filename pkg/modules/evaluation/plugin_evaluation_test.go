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
	require.Equal(t, 19, totalPlugins, "should load exactly 19 embedded plugins")

	// Verify plugins by category
	require.Contains(t, module.plugins, plugin.CategorySSH)
	require.Contains(t, module.plugins, plugin.CategoryHTTP)
	require.Contains(t, module.plugins, plugin.CategoryTLS)
	require.Contains(t, module.plugins, plugin.CategoryDatabase)
	require.Contains(t, module.plugins, plugin.CategoryNetwork)

	// Verify counts per category
	require.Len(t, module.plugins[plugin.CategorySSH], 5, "should have 5 SSH plugins")
	require.Len(t, module.plugins[plugin.CategoryHTTP], 4, "should have 4 HTTP plugins")
	require.Len(t, module.plugins[plugin.CategoryTLS], 4, "should have 4 TLS plugins")
	require.Len(t, module.plugins[plugin.CategoryDatabase], 3, "should have 3 Database plugins")
	require.Len(t, module.plugins[plugin.CategoryNetwork], 3, "should have 3 Network plugins")
}

func TestPluginEvaluationModule_Execute_WithContext(t *testing.T) {
	module := NewPluginEvaluationModule()
	require.NoError(t, module.Init("test-instance", nil))

	ctx := context.Background()

	// Provide input context that should match TLS weak protocol plugin
	inputs := map[string]interface{}{
		"tls.version":  "TLSv1.0", // Should match tls-weak-protocol plugin
		"service.port": 443,
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

	// Should have at least one vulnerability match (TLS weak protocol)
	require.NotEmpty(t, outputs, "should produce vulnerability outputs")

	// Verify vulnerability structure
	vuln, ok := outputs[0].Data.(VulnerabilityResult)
	require.True(t, ok, "output should be VulnerabilityResult")
	require.True(t, vuln.Matched, "vulnerability should be matched")
	require.NotEmpty(t, vuln.Plugin, "plugin name should be set")
	require.NotEmpty(t, vuln.Severity, "severity should be set")
	require.NotEmpty(t, vuln.Message, "message should be set")
	require.Equal(t, 443, vuln.Port, "port should match input")
}

func TestPluginEvaluationModule_Execute_NoContext(t *testing.T) {
	module := NewPluginEvaluationModule()
	require.NoError(t, module.Init("test-instance", nil))

	ctx := context.Background()
	inputs := map[string]interface{}{} // Empty context

	outputChan := make(chan engine.ModuleOutput, 10)
	done := make(chan struct{})

	var outputs []engine.ModuleOutput
	go func() {
		for output := range outputChan {
			outputs = append(outputs, output)
		}
		close(done)
	}()

	err := module.Execute(ctx, inputs, outputChan)
	close(outputChan)
	<-done

	require.NoError(t, err)
	require.Empty(t, outputs, "no context should produce no outputs")
}

func TestPluginEvaluationModule_Execute_TLSWeakCipher(t *testing.T) {
	module := NewPluginEvaluationModule()
	require.NoError(t, module.Init("test-instance", nil))

	ctx := context.Background()

	// Context that should match TLS weak cipher plugin
	inputs := map[string]interface{}{
		"tls.cipher_suites": []string{"TLS_RSA_WITH_DES_CBC_SHA"}, // Weak cipher
		"service.port":      443,
	}

	outputChan := make(chan engine.ModuleOutput, 10)
	done := make(chan struct{})

	var outputs []engine.ModuleOutput
	go func() {
		for output := range outputChan {
			outputs = append(outputs, output)
		}
		close(done)
	}()

	err := module.Execute(ctx, inputs, outputChan)
	close(outputChan)
	<-done

	require.NoError(t, err)
	require.NotEmpty(t, outputs, "should detect TLS weak cipher vulnerability")

	// Verify the match
	vuln, ok := outputs[0].Data.(VulnerabilityResult)
	require.True(t, ok)
	require.Contains(t, vuln.Plugin, "TLS")
	require.Equal(t, "high", vuln.Severity) // TLS weak cipher is high severity
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
