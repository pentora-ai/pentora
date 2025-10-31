package scanexec

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pentora-ai/pentora/pkg/appctx"
	"github.com/pentora-ai/pentora/pkg/engine"
	_ "github.com/pentora-ai/pentora/pkg/modules/discovery"
	_ "github.com/pentora-ai/pentora/pkg/modules/scan"
	"github.com/pentora-ai/pentora/pkg/storage"
)

// TestRun_HermeticLocal validates minimal execution path using an ephemeral
// localhost port and avoids any external environment dependencies.
func TestRun_HermeticLocal(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	// Use listener's host and actual port for a deterministic open-port scan.
	host, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)

	// Create a minimal AppManager using the factory and default config.
	factory := &engine.DefaultAppManagerFactory{}
	appMgr, err := factory.CreateWithNoConfig()
	require.NoError(t, err)
	ctx := context.WithValue(appMgr.Context(), engine.AppManagerKey, appMgr)
	ctx = appctx.WithConfig(ctx, appMgr.Config())

	svc := NewService()
	params := Params{
		Targets:       []string{host},
		OutputFormat:  "text",
		AllowLoopback: true,
		EnablePing:    false,
		Concurrency:   5,
		CustomTimeout: "300ms",
		Ports:         port,
	}

	res, _ := svc.Run(ctx, params)
	require.NotNil(t, res)
	require.NotEmpty(t, res.RunID)
}

// mock implementations to force branches in Service.Run
type mockPlanner struct {
	def *engine.DAGDefinition
	err error
}

func (m *mockPlanner) PlanDAG(intent engine.ScanIntent) (*engine.DAGDefinition, error) {
	return m.def, m.err
}

type mockOrch struct {
	out map[string]interface{}
	err error
}

func (m *mockOrch) Run(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
	return m.out, m.err
}

// minimal app manager impl for missing-context error path is handled by providing no manager

func TestRun_MissingAppManager(t *testing.T) {
	svc := NewService()
	_, err := svc.Run(context.Background(), Params{})
	require.Error(t, err)
}

func TestRun_PlannerInitError(t *testing.T) {
	factory := &engine.DefaultAppManagerFactory{}
	appMgr, err := factory.CreateWithNoConfig()
	require.NoError(t, err)
	ctx := context.WithValue(appMgr.Context(), engine.AppManagerKey, appMgr)
	ctx = appctx.WithConfig(ctx, appMgr.Config())

	svc := NewService().WithPlannerFactory(func() (dagPlanner, error) {
		return nil, errors.New("planner init fail")
	})
	_, e := svc.Run(ctx, Params{Targets: []string{"127.0.0.1"}})
	require.Error(t, e)
}

func TestRun_PlannerEmptyDAG(t *testing.T) {
	factory := &engine.DefaultAppManagerFactory{}
	appMgr, err := factory.CreateWithNoConfig()
	require.NoError(t, err)
	ctx := context.WithValue(appMgr.Context(), engine.AppManagerKey, appMgr)
	ctx = appctx.WithConfig(ctx, appMgr.Config())

	empty := &engine.DAGDefinition{Nodes: nil}
	svc := NewService().WithPlannerFactory(func() (dagPlanner, error) {
		return &mockPlanner{def: empty}, nil
	})
	_, e := svc.Run(ctx, Params{Targets: []string{"127.0.0.1"}})
	require.Error(t, e)
}

func TestRun_OrchestratorErrorAndStatus(t *testing.T) {
	factory := &engine.DefaultAppManagerFactory{}
	appMgr, err := factory.CreateWithNoConfig()
	require.NoError(t, err)
	ctx := context.WithValue(appMgr.Context(), engine.AppManagerKey, appMgr)
	ctx = appctx.WithConfig(ctx, appMgr.Config())

	def := &engine.DAGDefinition{Name: "test", Nodes: []engine.DAGNodeConfig{{InstanceID: "n1", ModuleType: "noop"}}}
	svc := NewService().
		WithPlannerFactory(func() (dagPlanner, error) { return &mockPlanner{def: def}, nil }).
		WithOrchestratorFactory(func(d *engine.DAGDefinition) (orchestrator, error) {
			return &mockOrch{out: map[string]interface{}{"discovery.live_hosts": []interface{}{}}, err: errors.New("run failed")}, nil
		})

	res, e := svc.Run(ctx, Params{Targets: []string{"127.0.0.1"}})
	require.Error(t, e)
	require.NotNil(t, res)
	require.Equal(t, "failed", res.Status)
}

// progress sink mock to capture emitted events
type capturingSink struct{ events []ProgressEvent }

func (c *capturingSink) OnEvent(e ProgressEvent) { c.events = append(c.events, e) }

// minimal in-memory scans store/backends for storage coverage
type memScans struct {
	created []*storage.ScanMetadata
	updates []struct {
		org string
		id  string
		upd storage.ScanUpdates
	}
}

func (m *memScans) Create(ctx context.Context, orgID string, meta *storage.ScanMetadata) error {
	m.created = append(m.created, meta)
	return nil
}

func (m *memScans) Get(ctx context.Context, orgID, scanID string) (*storage.ScanMetadata, error) {
	return nil, nil
}

func (m *memScans) Update(ctx context.Context, orgID, scanID string, upd storage.ScanUpdates) error {
	m.updates = append(m.updates, struct {
		org string
		id  string
		upd storage.ScanUpdates
	}{orgID, scanID, upd})
	return nil
}

func (m *memScans) List(ctx context.Context, orgID string, filter storage.ScanFilter) ([]*storage.ScanMetadata, error) {
	return nil, nil
}

func (m *memScans) Delete(ctx context.Context, orgID, scanID string) error { return nil }
func (m *memScans) ReadData(ctx context.Context, orgID, scanID string, dataType storage.DataType) (io.ReadCloser, error) {
	return nil, storage.ErrNotFound
}

func (m *memScans) WriteData(ctx context.Context, orgID, scanID string, dataType storage.DataType, data io.Reader) error {
	return nil
}

func (m *memScans) AppendData(ctx context.Context, orgID, scanID string, dataType storage.DataType, data []byte) error {
	return nil
}

func (m *memScans) GetAnalytics(ctx context.Context, orgID string, period storage.TimePeriod) (*storage.Analytics, error) {
	return nil, storage.ErrNotSupported
}

type memBackend struct{ scans *memScans }

func (b *memBackend) Scans() storage.ScanStore             { return b.scans }
func (b *memBackend) Initialize(ctx context.Context) error { return nil }
func (b *memBackend) Close() error                         { return nil }
func (b *memBackend) GarbageCollect(ctx context.Context, opts storage.GCOptions) (*storage.GCResult, error) {
	return &storage.GCResult{}, nil
}

func Test_WithProgressSink_And_WithStorage(t *testing.T) {
	factory := &engine.DefaultAppManagerFactory{}
	appMgr, err := factory.CreateWithNoConfig()
	require.NoError(t, err)
	ctx := context.WithValue(appMgr.Context(), engine.AppManagerKey, appMgr)
	ctx = appctx.WithConfig(ctx, appMgr.Config())

	def := &engine.DAGDefinition{Name: "test", Nodes: []engine.DAGNodeConfig{{InstanceID: "n1", ModuleType: "noop"}}}
	orchOut := map[string]interface{}{
		"discovery.live_hosts":  []interface{}{"127.0.0.1"},
		"scan.services":         []interface{}{map[string]interface{}{"port": 80}},
		"vulnerability.results": []interface{}{map[string]interface{}{"severity": "LOW"}},
	}

	// setup service with progress sink and storage
	sink := &capturingSink{}
	scans := &memScans{}
	backend := &memBackend{scans: scans}

	svc := NewService().
		WithProgressSink(sink).
		WithStorage(backend).
		WithPlannerFactory(func() (dagPlanner, error) { return &mockPlanner{def: def}, nil }).
		WithOrchestratorFactory(func(d *engine.DAGDefinition) (orchestrator, error) { return &mockOrch{out: orchOut, err: nil}, nil })

	res, runErr := svc.Run(ctx, Params{Targets: []string{"127.0.0.1"}})
	require.NoError(t, runErr)
	require.NotNil(t, res)
	require.Equal(t, "completed", res.Status)

	// progress sink received events
	require.GreaterOrEqual(t, len(sink.events), 2)

	// storage created once and updated at least once (status/stats)
	require.Equal(t, 1, len(scans.created))
	require.GreaterOrEqual(t, len(scans.updates), 1)
}
