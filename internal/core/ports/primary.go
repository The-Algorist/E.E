package ports

import (
	"context"
	"E.E/internal/core/domain"
)

// EncryptionService defines the primary port for encryption operations
type EncryptionService interface {
	// StartEncryption initiates the encryption process for a video
	StartEncryption(ctx context.Context, sourceURL string) (*domain.EncryptionJob, error)

	// GetJobStatus retrieves the current status of an encryption job
	GetJobStatus(ctx context.Context, jobID string) (*domain.EncryptionJob, error)

	// PauseJob pauses an ongoing encryption job
	PauseJob(ctx context.Context, jobID string) error

	// ResumeJob resumes a paused encryption job
	ResumeJob(ctx context.Context, jobID string) error

	// StopJob stops a specific encryption job
	StopJob(ctx context.Context, jobID string) error

	// StopEngine stops the entire encryption engine
	StopEngine() error

	// ListJobs returns a list of jobs with optional filtering and pagination
	ListJobs(ctx context.Context, limit, offset int, filter domain.JobFilter, sort domain.JobSort) ([]*domain.EncryptionJob, error)

	// GetJobsStatusSummary returns a summary of jobs grouped by status
	GetJobsStatusSummary(ctx context.Context) (map[string]interface{}, error)

	// Batch operations
	ProcessBatch(ctx context.Context, op domain.BatchOperation) (*domain.BatchResult, error)
	GetBatchResult(ctx context.Context, batchID string) (*domain.BatchResult, error)

	// Job history operations
	GetJobHistory(ctx context.Context, jobID string) ([]domain.JobHistoryEntry, error)
}

// EncryptionProgress represents a progress update channel
type EncryptionProgress interface {
	// UpdateProgress updates the progress of an encryption job
	UpdateProgress(jobID string, progress float64) error

	// SubscribeToProgress subscribes to progress updates for a job
	SubscribeToProgress(jobID string) (<-chan float64, error)
}