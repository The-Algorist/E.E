package domain

import "time"

// EncryptionStatus represents the current state of an encryption job
type EncryptionStatus string

const (
	StatusPending   EncryptionStatus = "PENDING"
	StatusProgress  EncryptionStatus = "IN_PROGRESS"
	StatusPaused    EncryptionStatus = "PAUSED"
	StatusCompleted EncryptionStatus = "COMPLETED"
	StatusFailed    EncryptionStatus = "FAILED"
)

// EncryptionJob represents an encryption task
type EncryptionJob struct {
	ID            string           `json:"id"`
	SourceURL     string          `json:"source_url"`
	Status        EncryptionStatus `json:"status"`
	Progress      float64         `json:"progress"`
	DecryptionKey string          `json:"decryption_key,omitempty"`
	Error         string          `json:"error,omitempty"`
	CreatedAt     int64           `json:"created_at"`
	UpdatedAt     int64           `json:"updated_at"`
}

// NewEncryptionJob creates a new encryption job
func NewEncryptionJob(sourceURL string) *EncryptionJob {
	now := time.Now().Unix()
	return &EncryptionJob{
		SourceURL: sourceURL,
		Status:   StatusPending,
		Progress: 0.0,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// EncryptionRequest represents the incoming request to start encryption
type EncryptionRequest struct {
	SourceURL string `json:"source_url,omitempty"`
	Batch     bool   `json:"batch,omitempty"`
	Action    BatchAction `json:"action,omitempty"`
	SourceURLs []string `json:"source_urls,omitempty"`
	JobIDs     []string `json:"job_ids,omitempty"`
}

// EncryptionResponse represents the response after starting encryption
type EncryptionResponse struct {
	JobID     string          `json:"job_id"`
	Status    EncryptionStatus `json:"status"`
	CreatedAt int64           `json:"created_at"`
}

// CanPause checks if the job can be paused
func (j *EncryptionJob) CanPause() error {
	switch j.Status {
	case StatusPaused:
		return NewJobStateError(j.ID, j.Status, "pause", "job is already paused")
	case StatusCompleted:
		return NewJobStateError(j.ID, j.Status, "pause", "cannot pause a completed job")
	case StatusFailed:
		return NewJobStateError(j.ID, j.Status, "pause", "cannot pause a failed job")
	case StatusPending:
		return NewJobStateError(j.ID, j.Status, "pause", "cannot pause a pending job")
	}
	return nil
}

// CanResume checks if the job can be resumed
func (j *EncryptionJob) CanResume() error {
	if j.Status != StatusPaused {
		return NewJobStateError(j.ID, j.Status, "resume", "can only resume paused jobs")
	}
	return nil
}

// CanStop checks if the job can be stopped
func (j *EncryptionJob) CanStop() error {
	switch j.Status {
	case StatusCompleted:
		return NewJobStateError(j.ID, j.Status, "stop", "job is already completed")
	case StatusFailed:
		return NewJobStateError(j.ID, j.Status, "stop", "job is already stopped")
	}
	return nil
}

// IsTerminal checks if the job is in a terminal state
func (j *EncryptionJob) IsTerminal() bool {
	return j.Status == StatusCompleted || j.Status == StatusFailed
}

// JobFilter contains all possible filtering options
type JobFilter struct {
	Status      string
	StartDate   int64  // Unix timestamp
	EndDate     int64  // Unix timestamp
	SourceURL   string
	MinProgress float64
}

// SortField represents a single sort criterion
type SortField struct {
	Field         string
	Order         string
	CaseSensitive bool
}

// JobSort represents sorting options
type JobSort struct {
	Fields []SortField
}
