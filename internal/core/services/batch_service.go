package services

import (
    "context"
    "fmt"
    "time"
    "github.com/google/uuid"

    "go.uber.org/zap"
    "E.E/internal/core/domain"
    "E.E/internal/core/ports"
)

type BatchService struct {
    encryptionService ports.EncryptionService
    jobRepository     ports.JobRepository
    batchRepository   ports.BatchRepository
    logger           *zap.Logger
}

func NewBatchService(
    encryptionService ports.EncryptionService,
    jobRepository ports.JobRepository,
    batchRepository ports.BatchRepository,
    logger *zap.Logger,
) *BatchService {
    return &BatchService{
        encryptionService: encryptionService,
        jobRepository:     jobRepository,
        batchRepository:   batchRepository,
        logger:           logger,
    }
}

// BatchValidationError represents validation errors with improved details
type BatchValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Value   string `json:"value,omitempty"`
}

func (s *BatchService) validateBatchOperation(op domain.BatchOperation) []BatchValidationError {
    var errors []BatchValidationError

    // Validate action
    if op.Action == "" {
        errors = append(errors, BatchValidationError{
            Field:   "action",
            Message: "action is required",
        })
    } else {
        // Validate action is supported
        validActions := map[domain.BatchAction]bool{
            domain.BatchActionStart:  true,
            domain.BatchActionPause:  true,
            domain.BatchActionResume: true,
            domain.BatchActionStop:   true,
            domain.BatchActionRetry:  true,
        }
        if !validActions[op.Action] {
            errors = append(errors, BatchValidationError{
                Field:   "action",
                Message: "unsupported action",
                Value:   string(op.Action),
            })
        }
    }

    // Action-specific validations
    switch op.Action {
    case domain.BatchActionStart:
        if len(op.SourceURLs) == 0 {
            errors = append(errors, BatchValidationError{
                Field:   "source_urls",
                Message: "at least one source URL is required for start action",
            })
        }
        // Validate each source URL
        for i, url := range op.SourceURLs {
            if url == "" {
                errors = append(errors, BatchValidationError{
                    Field:   fmt.Sprintf("source_urls[%d]", i),
                    Message: "source URL cannot be empty",
                })
            }
        }
        // Warn if job_ids are provided for start action
        if len(op.JobIDs) > 0 {
            errors = append(errors, BatchValidationError{
                Field:   "job_ids",
                Message: "job_ids should not be provided for start action",
                Value:   fmt.Sprintf("%v", op.JobIDs),
            })
        }

    case domain.BatchActionPause, domain.BatchActionResume, 
         domain.BatchActionStop, domain.BatchActionRetry:
        if len(op.JobIDs) == 0 {
            errors = append(errors, BatchValidationError{
                Field:   "job_ids",
                Message: fmt.Sprintf("at least one job ID is required for %s action", op.Action),
            })
        }
        // Validate each job ID
        for i, jobID := range op.JobIDs {
            if jobID == "" {
                errors = append(errors, BatchValidationError{
                    Field:   fmt.Sprintf("job_ids[%d]", i),
                    Message: "job ID cannot be empty",
                })
            }
        }
        // Warn if source_urls are provided for non-start actions
        if len(op.SourceURLs) > 0 {
            errors = append(errors, BatchValidationError{
                Field:   "source_urls",
                Message: fmt.Sprintf("source_urls should not be provided for %s action", op.Action),
                Value:   fmt.Sprintf("%v", op.SourceURLs),
            })
        }
    }

    return errors
}

func (s *BatchService) ProcessBatch(ctx context.Context, op domain.BatchOperation) (*domain.BatchResult, error) {
    // Validate batch operation
    if errors := s.validateBatchOperation(op); len(errors) > 0 {
        return nil, fmt.Errorf("validation failed: %v", errors)
    }

    result := &domain.BatchResult{
        BatchID:    generateBatchID(),
        StartTime:  time.Now(),
        Action:     op.Action,
        Successful: make([]string, 0),
        Failed:     make([]domain.BatchJobError, 0),
    }

    // Calculate total jobs based on action type
    var totalJobs int
    if op.Action == domain.BatchActionStart {
        totalJobs = len(op.SourceURLs)
    } else {
        totalJobs = len(op.JobIDs)
    }

    // Process the batch operation
    if op.Action == domain.BatchActionStart {
        for _, sourceURL := range op.SourceURLs {
            job, err := s.encryptionService.StartEncryption(ctx, sourceURL)
            if err != nil {
                result.Failed = append(result.Failed, domain.BatchJobError{
                    JobID: "N/A",
                    Error: fmt.Sprintf("Failed to create job for %s: %v", sourceURL, err),
                })
                continue
            }
            
            result.Successful = append(result.Successful, job.ID)
            
            // Add to job history
            historyEntry := domain.JobHistoryEntry{
                Timestamp: time.Now(),
                Action:    string(op.Action),
                BatchID:   result.BatchID,
                Status:    "created",
                Details: map[string]interface{}{
                    "batch_operation": true,
                    "batch_size":     totalJobs,
                },
            }
            
            if err := s.jobRepository.AddJobHistory(ctx, job.ID, historyEntry); err != nil {
                s.logger.Error("Failed to add job history entry",
                    zap.String("job_id", job.ID),
                    zap.String("batch_id", result.BatchID),
                    zap.Error(err))
            }
        }
    } else {
        // Process existing jobs
        for _, jobID := range op.JobIDs {
            err := s.processJob(ctx, jobID, op, 0)
            if err != nil {
                result.Failed = append(result.Failed, domain.BatchJobError{
                    JobID: jobID,
                    Error: err.Error(),
                })
            } else {
                result.Successful = append(result.Successful, jobID)
            }
        }
    }

    // Update and store result
    result.EndTime = time.Now()
    result.Summary = domain.BatchSummary{
        TotalJobs:    totalJobs,  // Use the calculated total
        SuccessCount: len(result.Successful),
        FailureCount: len(result.Failed),
        Duration:     result.EndTime.Sub(result.StartTime),
    }

    // Store batch result
    if err := s.batchRepository.StoreBatchResult(ctx, result); err != nil {
        s.logger.Error("Failed to store batch result",
            zap.String("batch_id", result.BatchID),
            zap.Error(err))
        return nil, fmt.Errorf("failed to store batch result: %w", err)
    }

    return result, nil
}

// Helper function to process individual job in batch
func (s *BatchService) processJob(ctx context.Context, jobID string, op domain.BatchOperation, index int) error {
    // First verify the job exists
    job, err := s.encryptionService.GetJobStatus(ctx, jobID)
    if err != nil {
        return fmt.Errorf("job not found: %s (error: %w)", jobID, err)
    }
    if job == nil {
        return fmt.Errorf("job %s does not exist", jobID)
    }

    // Enhanced action-specific validation and error messages
    switch op.Action {
    case domain.BatchActionStart:
        if index >= len(op.SourceURLs) {
            return fmt.Errorf("source URL index out of range for job %s", jobID)
        }
        _, err := s.encryptionService.StartEncryption(ctx, op.SourceURLs[index])
        if err != nil {
            return fmt.Errorf("failed to start encryption for job %s: %w", jobID, err)
        }
        return nil

    case domain.BatchActionPause:
        if err := job.CanPause(); err != nil {
            return fmt.Errorf("cannot pause job %s: %w", jobID, err)
        }
        if err := s.encryptionService.PauseJob(ctx, jobID); err != nil {
            return fmt.Errorf("failed to pause job %s: %w", jobID, err)
        }
        return nil

    case domain.BatchActionResume:
        if err := job.CanResume(); err != nil {
            return fmt.Errorf("cannot resume job %s: %w", jobID, err)
        }
        if err := s.encryptionService.ResumeJob(ctx, jobID); err != nil {
            return fmt.Errorf("failed to resume job %s: %w", jobID, err)
        }
        return nil

    case domain.BatchActionStop:
        if err := job.CanStop(); err != nil {
            return fmt.Errorf("cannot stop job %s: %w", jobID, err)
        }
        if err := s.encryptionService.StopJob(ctx, jobID); err != nil {
            return fmt.Errorf("failed to stop job %s: %w", jobID, err)
        }
        return nil

    case domain.BatchActionRetry:
        if job.Status != domain.StatusFailed {
            return fmt.Errorf("job %s is not in failed state (current status: %s)", jobID, job.Status)
        }
        _, err = s.encryptionService.StartEncryption(ctx, job.SourceURL)
        if err != nil {
            return fmt.Errorf("failed to retry job %s: %w", jobID, err)
        }
        return nil

    default:
        return fmt.Errorf("unsupported action %s for job %s", op.Action, jobID)
    }
}

func (s *BatchService) GetBatchResult(ctx context.Context, batchID string) (*domain.BatchResult, error) {
    return s.batchRepository.GetBatchResult(ctx, batchID)
}

func generateBatchID() string {
    return fmt.Sprintf("batch_%s", uuid.New().String())
}

func (s *BatchService) ListBatchResults(ctx context.Context, filter domain.BatchFilter) ([]*domain.BatchResult, error) {
    return s.batchRepository.ListBatchResults(ctx, filter)
}