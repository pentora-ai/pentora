package jobs

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

// MemoryManager is an in-memory job queue implementation.
// Suitable for single-instance OSS deployments.
type MemoryManager struct {
	concurrency int
	workers     []*worker
	queue       chan Job
	wg          sync.WaitGroup
	mu          sync.Mutex
}

// NewMemoryManager creates a new in-memory job manager.
func NewMemoryManager(concurrency int) *MemoryManager {
	if concurrency <= 0 {
		concurrency = 4
	}

	return &MemoryManager{
		concurrency: concurrency,
		queue:       make(chan Job, 100), // Buffer for 100 jobs
		workers:     make([]*worker, 0, concurrency),
	}
}

// Start launches worker goroutines.
func (m *MemoryManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Info().
		Str("component", "jobs").
		Int("workers", m.concurrency).
		Msg("Starting job manager")

	for i := 0; i < m.concurrency; i++ {
		w := &worker{
			id:    i,
			queue: m.queue,
		}
		m.workers = append(m.workers, w)

		m.wg.Add(1)
		go func(w *worker) {
			defer m.wg.Done()
			w.run(ctx)
		}(w)
	}

	log.Info().Msg("Job manager started")
	return nil
}

// Stop gracefully shuts down workers.
func (m *MemoryManager) Stop(ctx context.Context) error {
	log.Info().Msg("Stopping job manager...")

	// Close queue to signal workers
	close(m.queue)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("Job manager stopped gracefully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("job manager stop timeout: %w", ctx.Err())
	}
}

// worker processes jobs from the queue
type worker struct {
	id    int
	queue <-chan Job
}

func (w *worker) run(ctx context.Context) {
	log.Info().
		Str("component", "jobs").
		Int("worker_id", w.id).
		Msg("Worker started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Int("worker_id", w.id).Msg("Worker stopping (context cancelled)")
			return

		case job, ok := <-w.queue:
			if !ok {
				log.Info().Int("worker_id", w.id).Msg("Worker stopping (queue closed)")
				return
			}

			w.process(job)
		}
	}
}

func (w *worker) process(job Job) {
	log.Info().
		Str("component", "jobs").
		Int("worker_id", w.id).
		Str("job_id", job.ID).
		Str("job_type", job.Type).
		Msg("Processing job")

	// TODO: Actual job processing logic
	// For now, just log
}
