// pkg/scan/orchestrator.go
package scan

import (
	"context"
	"sync"

	"github.com/pentora-ai/pentora/pkg/hook"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/rs/zerolog/log"
)

// Orchestrator coordinates the execution of scan plugins.
type orchestrator struct {
	Registry plugin.RegistryInterface
	Hook     hook.Manager
}

// New creates a new ScanOrchestrator with the given plugin registry.
func New(reg plugin.RegistryInterface, h *hook.Manager) Orchestrator {
	return &orchestrator{
		Registry: reg,
		Hook:     *h,
	}
}

// RunAllPlugins runs all registered plugins for the given target.
func (o *orchestrator) RunAllPlugins(ctx context.Context, target string) {
	for _, p := range o.Registry.All() {
		scanID := GenerateScanID()
		pc := plugin.PluginContext{
			Target: target,
			ScanID: scanID,
			Logger: log.With().Str("plugin", p.Name()).Str("scan_id", scanID).Logger(),
		}

		if err := p.Run(ctx, pc); err != nil {
			pc.Logger.Err(err).Msg("plugin execution failed")
		} else {
			pc.Logger.Info().Msg("completed successfully")
		}
	}
}

// RunAllPluginsParallel runs all registered plugins concurrently for the given target.
func (o *orchestrator) RunAllPluginsParallel(ctx context.Context, target string) {
	var wg sync.WaitGroup
	plugins := o.Registry.All()

	for _, p := range plugins {
		wg.Add(1)

		go func(p plugin.Plugin) {
			defer wg.Done()

			scanID := GenerateScanID()
			pc := plugin.PluginContext{
				Target: target,
				ScanID: scanID,
				Logger: log.With().Str("plugin", p.Name()).Str("scan_id", scanID).Logger(),
			}

			if err := p.Run(ctx, pc); err != nil {
				pc.Logger.Err(err).Msg("plugin execution failed")
			} else {
				pc.Logger.Info().Msg("completed successfully")
			}
		}(p)
	}

	wg.Wait()
}

// RunPluginsDAG executes plugins in dependency-respecting order (sequential).
func (o *orchestrator) RunPluginsDAG(ctx context.Context, target string) error {
	// Build namedPlugin list
	var named []namedPlugin
	pluginMap := make(map[string]plugin.Plugin)

	for _, p := range o.Registry.All() {
		n := namedPlugin{
			Name:      p.Name(),
			DependsOn: p.DependsOn(),
		}
		named = append(named, n)
		pluginMap[n.Name] = p
	}

	// Sort by DAG
	sorted, err := topologicalSort(buildGraph(named))
	if err != nil {
		return err
	}

	// Run in sorted order
	for _, name := range sorted {
		p := pluginMap[name]
		scanID := GenerateScanID()
		pc := plugin.PluginContext{
			Target: target,
			ScanID: scanID,
			Logger: log.With().Str("plugin", name).Str("scan_id", scanID).Logger(),
		}

		if err := p.Run(ctx, pc); err != nil {
			pc.Logger.Err(err).Msg("plugin execution failed")
		} else {
			pc.Logger.Info().Msg("completed successfully")
		}
	}

	return nil
}

// RunPluginsDAGParallelLayers executes plugins respecting dependency layers, running each layer in parallel.
func (o *orchestrator) RunPluginsDAGParallelLayers(ctx context.Context, target string) error {
	// Prepare: plugin list and DAG graph
	var named []namedPlugin
	pluginMap := make(map[string]plugin.Plugin)

	for _, p := range o.Registry.All() {
		n := namedPlugin{
			Name:      p.Name(),
			DependsOn: p.DependsOn(),
		}
		named = append(named, n)
		pluginMap[n.Name] = p
	}

	graph := buildGraph(named)
	layers, err := dagLayers(graph)
	if err != nil {
		return err
	}

	// Iterate layer by layer, run each layer in parallel
	for _, layer := range layers {
		var wg sync.WaitGroup

		for _, name := range layer {
			p := pluginMap[name]
			wg.Add(1)

			go func(p plugin.Plugin) {
				defer wg.Done()

				scanID := GenerateScanID()
				pc := plugin.PluginContext{
					Target: target,
					ScanID: scanID,
					Logger: log.With().Str("plugin", p.Name()).Str("scan_id", scanID).Logger(),
					Hooks:  &o.Hook,
				}

				pc.Hooks.Trigger(ctx, "plugin:beforeRun:"+p.Name())

				if err := p.Run(ctx, pc); err != nil {
					pc.Hooks.Trigger(ctx, "plugin:onError:"+p.Name())
					pc.Logger.Err(err).Msg("plugin execution failed")
				} else {
					pc.Hooks.Trigger(ctx, "plugin:afterRun:"+p.Name())
					pc.Logger.Info().Msg("completed successfully")
				}
			}(p)
		}

		wg.Wait() // wait for the current layer to finish before moving to the next
	}

	return nil
}
