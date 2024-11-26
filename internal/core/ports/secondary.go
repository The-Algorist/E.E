package ports
import (
	"context"
	"io"

	"E.E/internal/core/domain"
)

// FileStorage defines the interface for file storage operations
type FileStorage interface {
	// ReadFile reads a file from storage
	ReadFile(path string) (io.ReadCloser, error)

	// WriteFile writes a file to storage
	WriteFile(path string, content io.Reader) error

	// DeleteFile removes a file from storage
	DeleteFile(path string) error

	// FileExists checks if a file exists in storage
	FileExists(path string) bool
}

// JobRepository defines the interface for job persistence operations
type JobRepository interface {
	// Create stores a new encryption job
	Create(ctx context.Context, job *domain.EncryptionJob) error

	// Update modifies an existing encryption job
	Update(ctx context.Context, job *domain.EncryptionJob) error

	// Get retrieves an encryption job by ID
	Get(ctx context.Context, jobID string) (*domain.EncryptionJob, error)

	// List retrieves all encryption jobs
	List(ctx context.Context) ([]*domain.EncryptionJob, error)

	// Delete removes an encryption job
	Delete(ctx context.Context, jobID string) error

	// HealthCheck checks the repository connection
	HealthCheck(ctx context.Context) error

	// Job history operations
	AddJobHistory(ctx context.Context, jobID string, entry domain.JobHistoryEntry) error
	GetJobHistory(ctx context.Context, jobID string) ([]domain.JobHistoryEntry, error)

	// Close closes the repository connection
	Close() error
}

// EncryptionEngine defines the interface for encryption operations
type EncryptionEngine interface {
	// Encrypt encrypts a file
	Encrypt(input io.Reader, output io.Writer) (string, error)

	// Decrypt decrypts a file
	Decrypt(input io.Reader, output io.Writer, key string) error

	// GenerateKey generates a new encryption key
	GenerateKey() (string, error)
}

// Add a new interface for batch operations persistence
type BatchRepository interface {
	// Store batch operation result
	StoreBatchResult(ctx context.Context, result *domain.BatchResult) error
	
	// Get batch operation result
	GetBatchResult(ctx context.Context, batchID string) (*domain.BatchResult, error)
	
	// List batch operations with optional filtering
	ListBatchResults(ctx context.Context, filter domain.BatchFilter) ([]*domain.BatchResult, error)
	
	// HealthCheck checks the repository connection
	HealthCheck(ctx context.Context) error

	// Close closes the repository connection
	Close() error
}