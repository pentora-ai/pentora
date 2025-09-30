package scanexec

import (
	"context"
	"fmt"
	"time"

	"github.com/pentora-ai/pentora/pkg/appctx"
	"github.com/pentora-ai/pentora/pkg/engine"
)

type dagPlanner interface {
	PlanDAG(intent engine.ScanIntent) (*engine.DAGDefinition, error)
}

// Service orchestrates scan execution using the engine planner/orchestrator.
type orchestrator interface {
	Run(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error)
}

type ProgressSink interface {
	OnEvent(ProgressEvent)
}

type ProgressEvent struct {
	Phase     string
	ModuleID  string
	Module    string
	Status    string
	Message   string
	Timestamp time.Time
}

type Service struct {
	plannerFactory      func() (dagPlanner, error)
	orchestratorFactory func(*engine.DAGDefinition) (orchestrator, error)
	progressSink        ProgressSink
}

// NewService builds a Service with default dependencies.
func NewService() *Service {
	return &Service{
		plannerFactory: func() (dagPlanner, error) {
			return engine.NewDAGPlanner(engine.GetRegisteredModuleFactories())
		},
		orchestratorFactory: func(def *engine.DAGDefinition) (orchestrator, error) {
			return engine.NewOrchestrator(def)
		},
	}
}

// WithProgressSink attaches a sink to receive progress notifications.
func (s *Service) WithProgressSink(sink ProgressSink) *Service {
	s.progressSink = sink
	return s
}

// WithPlannerFactory overrides planner construction for testing.
func (s *Service) WithPlannerFactory(factory func() (dagPlanner, error)) *Service {
	s.plannerFactory = factory
	return s
}

// WithOrchestratorFactory allows replacing the orchestrator constructor (useful for tests).
func (s *Service) WithOrchestratorFactory(factory func(*engine.DAGDefinition) (orchestrator, error)) *Service {
	s.orchestratorFactory = factory
	return s
}

// Run executes the scan pipeline using provided parameters and context carrying AppManager.
func (s *Service) Run(ctx context.Context, params Params) (*Result, error) {
	var appMgr engine.Manager
	switch v := ctx.Value(engine.AppManagerKey).(type) {
	case *engine.AppManager:
		appMgr = v
	case engine.Manager:
		appMgr = v
	default:
		return nil, fmt.Errorf("app manager missing from context")
	}

	if _, ok := appctx.Config(ctx); !ok {
		ctx = appctx.WithConfig(ctx, appMgr.Config())
	}

	planner, err := s.plannerFactory()
	if err != nil {
		return nil, fmt.Errorf("init planner: %w", err)
	}
	s.emit("plan", "", "planner", "start", "")

	intent := engine.ScanIntent{
		Targets:          params.Targets,
		Profile:          params.Profile,
		Level:            params.Level,
		IncludeTags:      params.IncludeTags,
		ExcludeTags:      params.ExcludeTags,
		EnableVulnChecks: params.EnableVuln,
		CustomPortConfig: params.Ports,
		CustomTimeout:    params.CustomTimeout,
		EnablePing:       params.EnablePing,
		PingCount:        params.PingCount,
		AllowLoopback:    params.AllowLoopback,
		Concurrency:      params.Concurrency,
		DiscoveryOnly:    params.OnlyDiscover,
		SkipDiscovery:    params.SkipDiscover,
	}
	if intent.DiscoveryOnly {
		intent.EnableVulnChecks = false
	}

	dagDefinition, err := planner.PlanDAG(intent)
	if err != nil {
		return nil, fmt.Errorf("plan dag: %w", err)
	}
	if dagDefinition == nil || len(dagDefinition.Nodes) == 0 {
		return nil, fmt.Errorf("planner produced empty dag")
	}
	s.emit("plan", "", "planner", "completed", fmt.Sprintf("nodes=%d", len(dagDefinition.Nodes)))

	orchestrator, err := s.orchestratorFactory(dagDefinition)
	if err != nil {
		return nil, fmt.Errorf("init orchestrator: %w", err)
	}

	inputs := map[string]interface{}{
		"config.targets":              params.Targets,
		"config.original_cli_targets": params.Targets,
		"config.output.format":        params.OutputFormat,
	}
	for k, v := range params.RawInputs {
		inputs[k] = v
	}

	s.emit("run", "", dagDefinition.Name, "start", "")
	dataCtx, runErr := orchestrator.Run(appMgr.Context(), inputs)
	status := statusFromError(runErr)
	s.emit("run", "", dagDefinition.Name, status, "")

	return &Result{
		Status:     status,
		Findings:   dataCtx,
		RawContext: dataCtx,
	}, runErr
}

func statusFromError(err error) string {
	if err != nil {
		return "failed"
	}
	return "completed"
}

func (s *Service) emit(phase, moduleID, module, status, msg string) {
	if s.progressSink == nil {
		return
	}
	s.progressSink.OnEvent(ProgressEvent{
		Phase:     phase,
		ModuleID:  moduleID,
		Module:    module,
		Status:    status,
		Message:   msg,
		Timestamp: time.Now(),
	})
}
