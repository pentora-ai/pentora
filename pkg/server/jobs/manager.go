package jobs

import "context"

// Manager defines the interface for background job processing.
// OSS implementation uses in-memory queue.
// Enterprise implementation can use distributed queue (Kafka, Redis, etc.)
type Manager interface {
	// Start begins processing jobs. Blocks until context is cancelled.
	Start(ctx context.Context) error

	// Stop gracefully shuts down job processing.
	// Waits for in-flight jobs to complete or context timeout.
	Stop(ctx context.Context) error

	// Submit adds a job to the queue (future implementation)
	// Submit(job Job) error

	// Status returns current queue statistics (future implementation)
	// Status() Status
}

// Job represents a unit of work to be processed
type Job struct {
	ID   string
	Type string
	Data map[string]interface{}
}

// Status holds job manager statistics
type Status struct {
	QueueDepth int
	ActiveJobs int
	Processed  int64
}
