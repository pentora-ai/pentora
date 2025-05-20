// pkg/scanner/ping.go
// Package scanner provides basic plugin implementations.
package scanner

import (
	"context"

	"github.com/pentora-ai/pentora/pkg/plugin"
)

// PingScanner is a dummy plugin that simulates a ping scan.
type PingScanner struct{}

// Name returns the plugin name.
func (p *PingScanner) Name() string {
	return "ping"
}

// Init is called once during application setup.
func (p *PingScanner) Init(ctx context.Context) error {
	// optional: warm-up, setup etc.
	return nil
}

// Run performs the actual scan logic.
func (p *PingScanner) Run(ctx context.Context, pc plugin.PluginContext) error {
	pc.Logger.Info().Msgf("pinging %s...", pc.Target)
	return nil
}

// Tags returns metadata tags for filtering.
func (p *PingScanner) Tags() []string {
	return []string{"network", "icmp", "passive"}
}

// DependsOn declares plugin dependencies.
func (p *PingScanner) DependsOn() []string {
	return []string{}
}

// Register the plugin to global registry
func init() {
	plugin.GlobalRegistry.Register(&PingScanner{})
}
