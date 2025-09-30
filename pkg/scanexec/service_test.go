package scanexec

import (
	"context"
	"errors"
	"testing"

	"github.com/pentora-ai/pentora/pkg/engine"
)

type stubPlanner struct {
	definition *engine.DAGDefinition
	err        error
}

func (s *stubPlanner) PlanDAG(intent engine.ScanIntent) (*engine.DAGDefinition, error) {
	return s.definition, s.err
}

type stubOrchestrator struct {
	outputs map[string]interface{}
	err     error
}

func (s *stubOrchestrator) Run(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
	if s.outputs == nil {
		s.outputs = make(map[string]interface{})
	}
	return s.outputs, s.err
}

func TestServiceRunMissingAppManager(t *testing.T) {
	svc := NewService()
	if _, err := svc.Run(context.Background(), Params{}); err == nil {
		t.Fatalf("expected error when app manager missing")
	}
}

func TestServicePlannerFailure(t *testing.T) {
	svc := NewService().WithPlannerFactory(func() (dagPlanner, error) {
		return nil, errors.New("boom")
	})

	ctx := context.WithValue(context.Background(), engine.AppManagerKey, &engine.AppManager{})

	if _, err := svc.Run(ctx, Params{}); err == nil {
		t.Fatalf("expected planner error")
	}
}

func TestServiceRunSuccess(t *testing.T) {
	planner := &stubPlanner{definition: &engine.DAGDefinition{Nodes: []engine.DAGNodeConfig{{InstanceID: "n1", ModuleType: "demo"}}}}
	stubOrch := &stubOrchestrator{outputs: map[string]interface{}{"demo": "ok"}}

	svc := NewService().
		WithPlannerFactory(func() (dagPlanner, error) { return planner, nil }).
		WithOrchestratorFactory(func(*engine.DAGDefinition) (orchestrator, error) { return stubOrch, nil })

	ctx := context.WithValue(context.Background(), engine.AppManagerKey, &engine.AppManager{})

	res, err := svc.Run(ctx, Params{Targets: []string{"127.0.0.1"}})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if res == nil || res.Status != "completed" {
		t.Fatalf("unexpected result: %#v", res)
	}
}
