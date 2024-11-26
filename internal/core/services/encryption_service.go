package services

import (
	"fmt"
	"time"
	"sort"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"context"
	"strings"

	"E.E/internal/core/domain"
	"E.E/internal/core/ports"
)

type EncryptionService struct {
	logger     *zap.Logger
	repository ports.JobRepository
	batchRepository ports.BatchRepository
}

func NewEncryptionService(repository ports.JobRepository, batchRepository ports.BatchRepository, logger *zap.Logger) ports.EncryptionService {
	return &EncryptionService{
		logger:     logger,
		repository: repository,
		batchRepository: batchRepository,
	}
}

// StartEncryption initiates an encryption job
func (s *EncryptionService) StartEncryption(ctx context.Context, sourceURL string) (*domain.EncryptionJob, error) {
	job := &domain.EncryptionJob{
		ID:        uuid.New().String(),
		SourceURL: sourceURL,
		Status:    domain.StatusProgress,
		Progress:  0.0,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	if err := s.repository.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return job, nil
}

// GetJobStatus retrieves the status of a job
func (s *EncryptionService) GetJobStatus(ctx context.Context, jobID string) (*domain.EncryptionJob, error) {
	job, err := s.repository.Get(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	return job, nil
}

// PauseJob simulates pausing an encryption job
func (s *EncryptionService) PauseJob(ctx context.Context, jobID string) error {
	s.logger.Info("Pausing encryption job", 
		zap.String("job_id", jobID),
		zap.String("status", string(domain.StatusPaused)),
	)
	return nil
}

// ResumeJob simulates resuming an encryption job
func (s *EncryptionService) ResumeJob(ctx context.Context, jobID string) error {
	s.logger.Info("Resuming encryption job", 
		zap.String("job_id", jobID),
		zap.String("status", string(domain.StatusProgress)),
	)
	return nil
}

// StopEngine is a killswitch to stop the encryption engine
func (s *EncryptionService) StopEngine() error {
	s.logger.Info("Stopping encryption engine")
	return nil
}

// StopJob simulates stopping a specific encryption job
func (s *EncryptionService) StopJob(ctx context.Context, jobID string) error {
	s.logger.Info("Stopping encryption job", 
		zap.String("job_id", jobID),
		zap.String("status", string(domain.StatusFailed)),
	)
	return nil
}

// ListJobs returns a list of jobs with filtering, sorting and pagination
func (s *EncryptionService) ListJobs(ctx context.Context, limit, offset int, filter domain.JobFilter, sortOpts domain.JobSort) ([]*domain.EncryptionJob, error) {
	// Validate sort options
	if err := validateSortOptions(sortOpts); err != nil {
		return nil, fmt.Errorf("invalid sort options: %w", err)
	}

	jobs, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	// Apply filters
	filtered := make([]*domain.EncryptionJob, 0)
	for _, job := range jobs {
		if !matchesFilter(job, filter) {
			continue
		}
		filtered = append(filtered, job)
	}

	// Apply sorting with enhanced options
	if err := sortJobs(filtered, sortOpts); err != nil {
		return nil, fmt.Errorf("failed to sort jobs: %w", err)
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(filtered) {
		return []*domain.EncryptionJob{}, nil
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

// GetJobsStatusSummary returns detailed statistics about jobs
func (s *EncryptionService) GetJobsStatusSummary(ctx context.Context) (map[string]interface{}, error) {
	jobs, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	summary := map[string]interface{}{
		"total": len(jobs),
		"by_status": map[string]int{
			string(domain.StatusPending):   0,
			string(domain.StatusProgress):  0,
			string(domain.StatusPaused):    0,
			string(domain.StatusCompleted): 0,
			string(domain.StatusFailed):    0,
		},
		"statistics": map[string]interface{}{
			"avg_completion_time": 0.0,
			"success_rate": 0.0,
			"total_completed": 0,
			"total_failed": 0,
			"avg_progress": 0.0,
			"jobs_last_24h": 0,
			"jobs_last_week": 0,
		},
		"latest_jobs": jobs[:min(5, len(jobs))],
	}

	var totalProgress float64
	var totalCompletionTime int64
	completedJobs := 0
	now := time.Now().Unix()
	dayAgo := now - 86400
	weekAgo := now - 604800

	for _, job := range jobs {
		// Count by status
		summary["by_status"].(map[string]int)[string(job.Status)]++
		
		// Calculate statistics
		totalProgress += job.Progress
		
		if job.Status == domain.StatusCompleted {
			completedJobs++
			totalCompletionTime += job.UpdatedAt - job.CreatedAt
		}

		// Count recent jobs
		if job.CreatedAt > dayAgo {
			summary["statistics"].(map[string]interface{})["jobs_last_24h"] = 
				summary["statistics"].(map[string]interface{})["jobs_last_24h"].(int) + 1
		}
		if job.CreatedAt > weekAgo {
			summary["statistics"].(map[string]interface{})["jobs_last_week"] = 
				summary["statistics"].(map[string]interface{})["jobs_last_week"].(int) + 1
		}
	}

	stats := summary["statistics"].(map[string]interface{})
	stats["avg_progress"] = totalProgress / float64(len(jobs))
	stats["total_completed"] = completedJobs
	stats["total_failed"] = summary["by_status"].(map[string]int)[string(domain.StatusFailed)]
	
	if completedJobs > 0 {
		stats["avg_completion_time"] = float64(totalCompletionTime) / float64(completedJobs)
	}
	if len(jobs) > 0 {
		stats["success_rate"] = float64(completedJobs) / float64(len(jobs)) * 100
	}

	return summary, nil
}

// Helper functions for filtering and sorting
func matchesFilter(job *domain.EncryptionJob, filter domain.JobFilter) bool {
	if filter.Status != "" && string(job.Status) != filter.Status {
		return false
	}
	if filter.StartDate > 0 && job.CreatedAt < filter.StartDate {
		return false
	}
	if filter.EndDate > 0 && job.CreatedAt > filter.EndDate {
		return false
	}
	if filter.SourceURL != "" && !strings.Contains(job.SourceURL, filter.SourceURL) {
		return false
	}
	if filter.MinProgress > 0 && job.Progress < filter.MinProgress {
		return false
	}
	return true
}

// Constants for sorting
const (
	// Sort Fields
	SortFieldCreatedAt = "created_at"  // Timestamp of job creation
	SortFieldUpdatedAt = "updated_at"  // Timestamp of last update
	SortFieldProgress  = "progress"    // Encryption progress (0-100)
	SortFieldStatus    = "status"      // Job status (PENDING, PROGRESS, etc.)
	SortFieldSourceURL = "source_url"  // Source URL of the video
	SortFieldID        = "id"          // Job ID (UUID)

	// Sort Orders
	SortOrderAsc  = "asc"   // Ascending order
	SortOrderDesc = "desc"  // Descending order

	// Maximum number of sort fields allowed
	MaxSortFields = 3
)

// GetAvailableSortOptions returns all available sorting options and their descriptions
func GetAvailableSortOptions() map[string]interface{} {
	return map[string]interface{}{
		"fields": map[string]string{
			SortFieldCreatedAt: "Timestamp when the job was created",
			SortFieldUpdatedAt: "Timestamp when the job was last updated",
			SortFieldProgress:  "Current progress of the encryption (0-100)",
			SortFieldStatus:    "Current status of the job",
			SortFieldSourceURL: "Source URL of the video being encrypted",
			SortFieldID:        "Unique identifier of the job",
		},
		"orders": map[string]string{
			SortOrderAsc:  "Ascending order (A-Z, 0-9, oldest first)",
			SortOrderDesc: "Descending order (Z-A, 9-0, newest first)",
		},
		"features": map[string]interface{}{
			"max_sort_fields":     MaxSortFields,
			"case_sensitive":      "Optional per-field setting (default: false)",
			"default_sort":        "created_at desc",
			"default_order":       "asc",
			"stable_sort":         true,
			"null_values_handled": true,
		},
		"examples": []map[string]string{
			{
				"description": "Sort by status ascending",
				"url":        "/api/v1/jobs?sort_by=status&order=asc",
			},
			{
				"description": "Sort by progress desc, then creation date",
				"url":        "/api/v1/jobs?sort_by=progress&sort_by=created_at&order=desc&order=asc",
			},
			{
				"description": "Case-sensitive URL sorting",
				"url":        "/api/v1/jobs?sort_by=source_url&case_sensitive=true",
			},
		},
	}
}

// Update validation for sort options
func validateSortOptions(sortOpts domain.JobSort) error {
	validFields := map[string]bool{
		SortFieldCreatedAt: true,
		SortFieldUpdatedAt: true,
		SortFieldProgress:  true,
		SortFieldStatus:    true,
		SortFieldSourceURL: true,
		SortFieldID:       true,
	}

	for _, field := range sortOpts.Fields {
		if field.Field != "" {
			if !validFields[strings.ToLower(field.Field)] {
				return fmt.Errorf("invalid sort field: %s. valid fields are: created_at, updated_at, progress, status, source_url, id", 
					field.Field)
			}
		}

		if field.Order != "" {
			order := strings.ToLower(field.Order)
			if order != SortOrderAsc && order != SortOrderDesc {
				return fmt.Errorf("invalid sort order: %s. valid orders are: asc, desc", field.Order)
			}
		}
	}

	return nil
}

// Enhanced sortJobs with more options and error handling
func sortJobs(jobs []*domain.EncryptionJob, sortOpts domain.JobSort) error {
	if len(sortOpts.Fields) == 0 {
		// Default sort: created_at desc, case-insensitive
		sortOpts.Fields = []domain.SortField{{
			Field: SortFieldCreatedAt,
			Order: SortOrderDesc,
			CaseSensitive: false,
		}}
	}

	sort.SliceStable(jobs, func(i, j int) bool {
		// Compare each sort field in order until a difference is found
		for _, field := range sortOpts.Fields {
			field.Field = strings.ToLower(field.Field)
			field.Order = strings.ToLower(field.Order)
			
			if field.Order == "" {
				field.Order = SortOrderAsc
			}

			var comparison int
			switch field.Field {
			case SortFieldCreatedAt, SortFieldUpdatedAt, SortFieldProgress:
				// Numeric fields - case sensitivity doesn't apply
				comparison = compareValues(jobs[i], jobs[j], field.Field)
			case SortFieldStatus, SortFieldSourceURL:
				// String fields - apply case sensitivity setting
				comparison = compareStrings(
					getFieldValue(jobs[i], field.Field),
					getFieldValue(jobs[j], field.Field),
					field.CaseSensitive,
				)
			case SortFieldID:
				// IDs should always be case-sensitive
				comparison = strings.Compare(jobs[i].ID, jobs[j].ID)
			}

			if comparison != 0 {
				return field.Order == SortOrderAsc && comparison < 0 ||
					   field.Order == SortOrderDesc && comparison > 0
			}
		}
		return false
	})

	return nil
}

// Helper function to compare string values
func compareStrings(a, b string, caseSensitive bool) int {
	if !caseSensitive {
		a = strings.ToLower(a)
		b = strings.ToLower(b)
	}
	return strings.Compare(a, b)
}

// Helper function to get field value as string
func getFieldValue(job *domain.EncryptionJob, field string) string {
	switch field {
	case SortFieldStatus:
		return string(job.Status)
	case SortFieldSourceURL:
		return job.SourceURL
	default:
		return ""
	}
}

// Helper function to compare numeric values
func compareValues(a, b *domain.EncryptionJob, field string) int {
	switch field {
	case SortFieldCreatedAt:
		return compareInt64(a.CreatedAt, b.CreatedAt)
	case SortFieldUpdatedAt:
		return compareInt64(a.UpdatedAt, b.UpdatedAt)
	case SortFieldProgress:
		return compareFloat64(a.Progress, b.Progress)
	default:
		return 0
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Add these helper functions
func compareInt64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareFloat64(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// ProcessBatch handles batch operations
func (s *EncryptionService) ProcessBatch(ctx context.Context, op domain.BatchOperation) (*domain.BatchResult, error) {
	s.logger.Info("Processing batch operation", 
		zap.String("action", string(op.Action)),
		zap.Int("job_count", len(op.JobIDs)))
	
	batchService := NewBatchService(s, s.repository, s.batchRepository, s.logger)
	return batchService.ProcessBatch(ctx, op)
}

// GetBatchResult retrieves a batch operation result
func (s *EncryptionService) GetBatchResult(ctx context.Context, batchID string) (*domain.BatchResult, error) {
	return s.batchRepository.GetBatchResult(ctx, batchID)
}

// GetJobHistory retrieves job history
func (s *EncryptionService) GetJobHistory(ctx context.Context, jobID string) ([]domain.JobHistoryEntry, error) {
	return s.repository.GetJobHistory(ctx, jobID)
}