package app

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/server/api"
	"github.com/pentora-ai/pentora/pkg/server/httpx"
	"github.com/pentora-ai/pentora/pkg/server/jobs"
	"github.com/rs/zerolog/log"
)

// App orchestrates the server runtime components:
// - HTTP server (API + UI)
// - Background job manager
// - Lifecycle management
type App struct {
	HTTP   *http.Server
	Jobs   jobs.Manager
	Ready  *atomic.Bool
	Config config.ServerConfig
	Deps   *Deps
}

// New creates and configures a new server application.
func New(ctx context.Context, cfg config.ServerConfig, deps *Deps) (*App, error) {
	log.Info().
		Str("component", "app").
		Msg("Initializing server application")

	// Initialize storage backend if provided
	if deps.Storage != nil {
		log.Info().
			Str("component", "app").
			Msg("Initializing storage backend")
		if err := deps.Storage.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize storage: %w", err)
		}
	}

	// Prepare API dependencies
	ready := &atomic.Bool{}
	apiDeps := &api.Deps{
		Storage:       deps.Storage,
		Workspace:     deps.Workspace,
		PluginService: deps.PluginService,
		Ready:         ready,
	}

	// Create router with all endpoints mounted
	router := httpx.NewRouter(cfg, apiDeps)

	if cfg.APIEnabled {
		log.Info().
			Str("component", "app").
			Msg("API endpoints enabled")
	} else {
		log.Warn().
			Str("component", "app").
			Msg("API endpoints disabled")
	}

	// UI handler is already mounted in router.NewRouter()
	if !cfg.UIEnabled {
		log.Warn().
			Str("component", "app").
			Msg("UI serving disabled")
	}

	// Create HTTP server with middleware
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Addr, cfg.Port),
		Handler:      httpx.Chain(cfg, router),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// Create job manager
	var jobsMgr jobs.Manager
	if cfg.JobsEnabled {
		jobsMgr = jobs.NewMemoryManager(cfg.Concurrency)
	}

	return &App{
		HTTP:   httpServer,
		Jobs:   jobsMgr,
		Ready:  ready,
		Config: cfg,
		Deps:   deps,
	}, nil
}

// Run starts the server and blocks until shutdown.
func (a *App) Run(ctx context.Context) error {
	log.Info().
		Str("component", "app").
		Str("addr", a.HTTP.Addr).
		Bool("api", a.Config.APIEnabled).
		Bool("ui", a.Config.UIEnabled).
		Bool("jobs", a.Config.JobsEnabled).
		Msg("Starting Pentora server")

	// Start HTTP server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := a.HTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}()

	// Start background jobs
	if a.Config.JobsEnabled && a.Jobs != nil {
		if err := a.Jobs.Start(ctx); err != nil {
			return fmt.Errorf("start jobs: %w", err)
		}
	}

	// Mark as ready
	a.Ready.Store(true)
	log.Info().
		Str("component", "app").
		Msg("Server is ready and accepting connections")

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		log.Info().
			Str("component", "app").
			Msg("Shutdown signal received")
	case err := <-serverErr:
		log.Error().
			Str("component", "app").
			Err(err).
			Msg("Server error")
		return err
	}

	// Graceful shutdown
	return a.shutdown()
}

// shutdown performs graceful shutdown of all components.
func (a *App) shutdown() error {
	log.Info().
		Str("component", "app").
		Msg("Initiating graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Mark as not ready
	a.Ready.Store(false)

	// Shutdown HTTP server
	log.Info().
		Str("component", "app").
		Msg("Shutting down HTTP server...")
	if err := a.HTTP.Shutdown(shutdownCtx); err != nil {
		log.Error().
			Str("component", "app").
			Err(err).
			Msg("HTTP server shutdown failed")
		return err
	}
	log.Info().
		Str("component", "app").
		Msg("HTTP server stopped")

	// Stop background jobs
	if a.Config.JobsEnabled && a.Jobs != nil {
		log.Info().
			Str("component", "app").
			Msg("Stopping background jobs...")
		if err := a.Jobs.Stop(shutdownCtx); err != nil {
			log.Error().
				Str("component", "app").
				Err(err).
				Msg("Jobs shutdown failed")
			return err
		}
		log.Info().
			Str("component", "app").
			Msg("Background jobs stopped")
	}

	// Close storage backend
	if a.Deps.Storage != nil {
		log.Info().
			Str("component", "app").
			Msg("Closing storage backend...")
		if err := a.Deps.Storage.Close(); err != nil {
			log.Error().
				Str("component", "app").
				Err(err).
				Msg("Storage close failed")
			return err
		}
		log.Info().
			Str("component", "app").
			Msg("Storage backend closed")
	}

	log.Info().
		Str("component", "app").
		Msg("Server shutdown complete")
	return nil
}
