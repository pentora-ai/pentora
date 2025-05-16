// pkg/scan/interface.go
package scan

import "context"

// Orchestrator defines the interface for plugin execution orchestration.
type Orchestrator interface {
	RunAllPlugins(ctx context.Context, target string)
	RunAllPluginsParallel(ctx context.Context, target string)
	RunPluginsDAG(ctx context.Context, target string) error
	RunPluginsDAGParallelLayers(ctx context.Context, target string) error
}
