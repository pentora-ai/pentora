package appctx

import (
	"context"

	"github.com/pentora-ai/pentora/pkg/config"
)

type key string

const configKey key = "pentora.config.manager"

// WithConfig stores the shared config manager on context.
func WithConfig(ctx context.Context, manager *config.Manager) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, configKey, manager)
}

// Config retrieves the shared config manager from context.
func Config(ctx context.Context) (*config.Manager, bool) {
	if ctx == nil {
		return nil, false
	}
	mgr, ok := ctx.Value(configKey).(*config.Manager)
	return mgr, ok && mgr != nil
}
