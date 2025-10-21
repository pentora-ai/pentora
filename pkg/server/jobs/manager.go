// pkg/server/jobs/manager.go
package jobs

import (
	"context"
)

// Job represents a background job to be executed.
type Job struct {
	ID      string
	Type    string
	Payload interface{}
}

// Status represents the current status of a job.
type Status struct {
	ID        string
	State     string // pending, running, completed, failed
	Error     error
	Progress  int // 0-100
	StartedAt int64
	EndedAt   int64
}

// Manager defines the interface for background job processing.
// OSS uses in-memory implementation; Enterprise can provide distributed queue.
type Manager interface {
	// Start begins processing jobs in the background.
	// It should be non-blocking and return immediately after starting workers.
	Start(ctx context.Context) error

	// Stop gracefully stops all workers and waits for in-flight jobs to complete.
	// It should respect the context deadline for shutdown timeout.
	Stop(ctx context.Context) error

	// Submit enqueues a job for processing (optional for MVP).
	// Submit(ctx context.Context, job Job) error

	// Status returns the current status of a job (optional for MVP).
	// Status(ctx context.Context, jobID string) (*Status, error)
}
