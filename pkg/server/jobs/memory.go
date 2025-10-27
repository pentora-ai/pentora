// pkg/server/jobs/memory.go
package jobs

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

// MemoryManager is an in-memory implementation of Manager for OSS.
// It processes jobs using a worker pool with configurable concurrency.
type MemoryManager struct {
	concurrency int
	jobQueue    chan Job
	wg          sync.WaitGroup
	cancelFunc  context.CancelFunc
	mu          sync.RWMutex
	started     bool
}

// NewMemoryManager creates a new in-memory job manager.
// concurrency controls the number of worker goroutines.
// If concurrency <= 0, defaults to 4.
func NewMemoryManager(concurrency int) *MemoryManager {
	if concurrency <= 0 {
		concurrency = 4 // Default worker count
	}

	return &MemoryManager{
		concurrency: concurrency,
		jobQueue:    make(chan Job, 100), // Buffered channel for jobs
		started:     false,
	}
}

// Start begins processing jobs in the background.
// It spawns worker goroutines that process jobs from the queue.
func (m *MemoryManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("job manager already started")
	}

	// Create cancellable context for workers
	workerCtx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel

	// Start worker pool
	for i := 0; i < m.concurrency; i++ {
		m.wg.Add(1)
		go m.worker(workerCtx, i)
	}

	m.started = true
	log.Info().
		Str("component", "jobs").
		Int("workers", m.concurrency).
		Msg("Job manager started")

	return nil
}

// Stop gracefully stops all workers and waits for in-flight jobs to complete.
// It respects the context deadline for shutdown timeout.
func (m *MemoryManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return nil // Already stopped
	}

	// Signal workers to stop
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	m.started = false
	m.mu.Unlock()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().
			Str("component", "jobs").
			Msg("Job manager stopped gracefully")
		return nil
	case <-ctx.Done():
		log.Warn().
			Str("component", "jobs").
			Msg("Job manager shutdown timed out")
		return ctx.Err()
	}
}

// worker processes jobs from the queue until the context is canceled.
func (m *MemoryManager) worker(ctx context.Context, id int) {
	defer m.wg.Done()

	log.Debug().
		Str("component", "jobs").
		Int("worker_id", id).
		Msg("Worker started")

	for {
		select {
		case <-ctx.Done():
			log.Debug().
				Str("component", "jobs").
				Int("worker_id", id).
				Msg("Worker stopping")
			return
		case job := <-m.jobQueue:
			// Process job (placeholder for MVP)
			log.Debug().
				Str("component", "jobs").
				Int("worker_id", id).
				Str("job_id", job.ID).
				Str("job_type", job.Type).
				Msg("Processing job")
			// Actual job processing logic would go here
		}
	}
}
