package scanexec

import (
    "context"
    "net"
    "testing"

    "github.com/stretchr/testify/require"

    "github.com/pentora-ai/pentora/pkg/appctx"
    "github.com/pentora-ai/pentora/pkg/engine"
    _ "github.com/pentora-ai/pentora/pkg/modules/discovery"
    _ "github.com/pentora-ai/pentora/pkg/modules/scan"
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
