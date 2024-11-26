package domain

import "time"

// BatchFilter defines filtering options for batch operations
type BatchFilter struct {
    StartTime    *time.Time  `json:"start_time,omitempty"`    // Filter by start time
    EndTime      *time.Time  `json:"end_time,omitempty"`      // Filter by end time
    Action       BatchAction `json:"action,omitempty"`         // Filter by action type
    Status       string      `json:"status,omitempty"`         // Filter by batch status
    MinSuccess   *int        `json:"min_success,omitempty"`    // Filter by minimum successful jobs
    MaxFailures  *int        `json:"max_failures,omitempty"`   // Filter by maximum failed jobs
    JobIDs       []string    `json:"job_ids,omitempty"`       // Filter by specific job IDs
}

// BatchOperation represents a batch action request
type BatchOperation struct {
    JobIDs     []string    `json:"job_ids"`
    Action     BatchAction `json:"action"`
    SourceURLs []string    `json:"source_urls,omitempty"`
}

type BatchAction string

const (
    BatchActionStart  BatchAction = "start"
    BatchActionPause  BatchAction = "pause"
    BatchActionResume BatchAction = "resume"
    BatchActionStop   BatchAction = "stop"
    BatchActionRetry  BatchAction = "retry"
)

// BatchResult represents the outcome of a batch operation
type BatchResult struct {
    BatchID    string         `json:"batch_id"`
    StartTime  time.Time      `json:"start_time"`
    EndTime    time.Time      `json:"end_time"`
    Action     BatchAction    `json:"action"`
    Successful []string       `json:"successful"`
    Failed     []BatchJobError `json:"failed"`
    Summary    BatchSummary   `json:"summary"`
}

type BatchJobError struct {
    JobID string `json:"job_id"`
    Error string `json:"error"`
}

type BatchSummary struct {
    TotalJobs    int           `json:"total_jobs"`
    SuccessCount int           `json:"success_count"`
    FailureCount int           `json:"failure_count"`
    Duration     time.Duration `json:"duration"`
}

// JobHistoryEntry represents a single history entry for a job
type JobHistoryEntry struct {
    Timestamp time.Time                 `json:"timestamp"`
    Action    string                    `json:"action"`
    Status    string                    `json:"status"`
    BatchID   string                    `json:"batch_id,omitempty"`
    Error     string                    `json:"error,omitempty"`
    Details   map[string]interface{}    `json:"details,omitempty"`
}