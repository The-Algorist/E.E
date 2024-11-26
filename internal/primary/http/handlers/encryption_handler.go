package handlers

import (
	// "net/http"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
	"time"
	"errors"
	"strings"
	"net/http"
	
	"E.E/internal/core/domain"
	"E.E/internal/core/ports"
	"E.E/internal/core/services"
)

type EncryptionHandler struct {
	encryptionService ports.EncryptionService
	logger           *zap.Logger
	errorHandler     *ErrorHandler
}

func NewEncryptionHandler(service ports.EncryptionService, logger *zap.Logger) *EncryptionHandler {
	return &EncryptionHandler{
		encryptionService: service,
		logger:           logger,
		errorHandler:     NewErrorHandler(logger),
	}
}

// StartEncryption handles the request to start video encryption
func (h *EncryptionHandler) StartEncryption(c *gin.Context) {
	var req domain.EncryptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorHandler.HandleError(c,
			domain.StatusBadRequest,
			"Invalid request format",
			[]domain.BatchError{{
				Field:   "request",
				Message: err.Error(),
				Code:    domain.ErrCodeInvalidFormat,
			}},
		)
		return
	}

	if req.Batch {
		h.handleBatchEncryption(c, req)
		return
	}

	h.handleSingleEncryption(c, req)
}

func (h *EncryptionHandler) handleBatchEncryption(c *gin.Context, req domain.EncryptionRequest) {
	op := domain.BatchOperation{
		Action:     req.Action,
		SourceURLs: req.SourceURLs,
		JobIDs:     req.JobIDs,
	}
	
	result, err := h.encryptionService.ProcessBatch(c.Request.Context(), op)
	if err != nil {
		var jobStateErr *domain.JobStateError
		if errors.As(err, &jobStateErr) {
			h.errorHandler.HandleStateError(c, jobStateErr)
			return
		}
		
		h.errorHandler.HandleBatchError(c,
			domain.StatusBadRequest,
			"Failed to process batch operation",
			[]domain.BatchError{{
				Field:      "batch",
				Message:    err.Error(),
				Code:       domain.ErrCodeBatchOperation,
				ActionType: string(op.Action),
			}},
			&domain.BatchDetails{
				Action:     string(op.Action),
				JobIDs:     op.JobIDs,
				SourceURLs: op.SourceURLs,
			},
		)
		return
	}
	
	c.JSON(domain.StatusAccepted, result)
}

func (h *EncryptionHandler) handleSingleEncryption(c *gin.Context, req domain.EncryptionRequest) {
	if req.SourceURL == "" {
		h.errorHandler.HandleError(c,
			domain.StatusBadRequest,
			"Validation error",
			[]domain.BatchError{
				domain.NewValidationError("source_url", "source_url is required for single operations", ""),
			},
		)
		return
	}

	job, err := h.encryptionService.StartEncryption(c.Request.Context(), req.SourceURL)
	if err != nil {
		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to start encryption",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeEncryptionFailed,
			}},
		)
		return
	}

	c.JSON(domain.StatusAccepted, domain.EncryptionResponse{
		JobID:     job.ID,
		Status:    job.Status,
		CreatedAt: job.CreatedAt,
	})
}

// GetStatus handles the request to check encryption status
func (h *EncryptionHandler) GetStatus(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		h.errorHandler.HandleError(c,
			domain.StatusBadRequest,
			"Validation error",
			[]domain.BatchError{domain.NewValidationError("job_id", "job_id is required", "")},
		)
		return
	}

	job, err := h.encryptionService.GetJobStatus(c.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			h.errorHandler.HandleError(c,
				domain.StatusNotFound,
				"Job not found",
				[]domain.BatchError{domain.NewNotFoundError("job", jobID)},
			)
			return
		}

		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to get job status",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeEncryptionFailed,
			}},
		)
		return
	}

	c.JSON(domain.StatusOK, job)
}

// ListJobs handles the request to list all jobs
func (h *EncryptionHandler) ListJobs(c *gin.Context) {
	// Pagination
	limit := 10
	offset := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Filtering
	filter := domain.JobFilter{
		Status:      c.Query("status"),
		SourceURL:   c.Query("source_url"),
		MinProgress: parseFloat(c.Query("min_progress"), 0),
	}
	if startDate := c.Query("start_date"); startDate != "" {
		filter.StartDate = parseTimestamp(startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		filter.EndDate = parseTimestamp(endDate)
	}

	// Parse sort fields with case-insensitive as default
	var sortFields []domain.SortField
	sortBy := c.QueryArray("sort_by")
	sortOrder := c.QueryArray("order")
	caseSensitive := c.QueryArray("case_sensitive")

	for i, field := range sortBy {
		sortField := domain.SortField{
			Field: field,
			CaseSensitive: false, // default to case-insensitive
		}

		// Get corresponding order if available
		if i < len(sortOrder) {
			sortField.Order = sortOrder[i]
		}

		// Only make case-sensitive if explicitly requested
		if i < len(caseSensitive) && caseSensitive[i] == "true" {
			sortField.CaseSensitive = true
		}

		sortFields = append(sortFields, sortField)
	}

	sort := domain.JobSort{
		Fields: sortFields,
	}

	ctx := c.Request.Context()
	jobs, err := h.encryptionService.ListJobs(ctx, limit, offset, filter, sort)
	if err != nil {
		if strings.Contains(err.Error(), "invalid sort") {
			errResp := domain.NewBatchErrorResponse(
				"Invalid sort parameters",
				[]domain.BatchError{
					domain.NewValidationError("sort", err.Error(), sort.Fields[0].Field),
				},
				nil,
				"",
			)
			c.JSON(domain.StatusBadRequest, errResp)
			return
		}
		
		h.logger.Error("Failed to list jobs", zap.Error(err))
		errResp, status := domain.GetBatchErrorResponse(err, "list")
		c.JSON(status, errResp)
		return
	}

	c.JSON(domain.StatusOK, gin.H{
		"jobs":       jobs,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  len(jobs),
		},
		"filter":       filter,
		"sort":        sort,
		"sort_options": services.GetAvailableSortOptions(),
	})
}

// JobsStatus returns a summary of all jobs grouped by status
func (h *EncryptionHandler) JobsStatus(c *gin.Context) {
	ctx := c.Request.Context()
	summary, err := h.encryptionService.GetJobsStatusSummary(ctx)
	if err != nil {
		h.logger.Error("Failed to get jobs status summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get jobs status summary",
		})
		return
	}

	response := domain.JobStatusSummaryResponse{
		Summary:   summary,
		Timestamp: time.Now().Unix(),
		Message:   "Job status summary retrieved successfully",
	}
	c.JSON(domain.StatusOK, response)
}

// ProcessBatch handles the request to process a batch of encryption jobs
func (h *EncryptionHandler) ProcessBatch(c *gin.Context) {
	var op domain.BatchOperation
	if err := c.ShouldBindJSON(&op); err != nil {
		h.errorHandler.HandleError(c,
			domain.StatusBadRequest,
			"Invalid request format",
			[]domain.BatchError{{
				Field:   "request",
				Message: err.Error(),
				Code:    domain.ErrCodeInvalidFormat,
			}},
		)
		return
	}

	result, err := h.encryptionService.ProcessBatch(c.Request.Context(), op)
	if err != nil {
		h.errorHandler.HandleBatchError(c,
			domain.StatusInternalServerError,
			"Failed to process batch operation",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeBatchOperation,
			}},
			&domain.BatchDetails{
				Action: string(op.Action),
				JobIDs: op.JobIDs,
			},
		)
		return
	}

	c.JSON(domain.StatusOK, result)
}

// GetBatchResult handles the request to retrieve a batch operation result
func (h *EncryptionHandler) GetBatchResult(c *gin.Context) {
	batchID := c.Param("batchId")
	if batchID == "" {
		h.errorHandler.HandleError(c,
			domain.StatusBadRequest,
			"Validation error",
			[]domain.BatchError{domain.NewValidationError("batch_id", "batch_id is required", "")},
		)
		return
	}

	result, err := h.encryptionService.GetBatchResult(c.Request.Context(), batchID)
	if err != nil {
		if errors.Is(err, domain.ErrBatchNotFound) {
			h.errorHandler.HandleError(c,
				domain.StatusNotFound,
				"Batch not found",
				[]domain.BatchError{domain.NewNotFoundError("batch", batchID)},
			)
			return
		}

		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to get batch result",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeBatchOperation,
			}},
		)
		return
	}

	c.JSON(domain.StatusOK, gin.H{
		"batch_id":  batchID,
		"result":    result,
		"timestamp": time.Now().Unix(),
		"message":   "Batch result retrieved successfully",
	})
}

// GetJobHistory handles the request to retrieve job history
func (h *EncryptionHandler) GetJobHistory(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		h.errorHandler.HandleError(c,
			domain.StatusBadRequest,
			"Validation error",
			[]domain.BatchError{domain.NewValidationError("job_id", "job_id is required", "")},
		)
		return
	}

	history, err := h.encryptionService.GetJobHistory(c.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			h.errorHandler.HandleError(c,
				domain.StatusNotFound,
				"Job not found",
				[]domain.BatchError{domain.NewNotFoundError("job", jobID)},
			)
			return
		}

		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to get job history",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeEncryptionFailed,
			}},
		)
		return
	}

	c.JSON(domain.StatusOK, history)
}

// PauseJob handles the request to pause an encryption job
func (h *EncryptionHandler) PauseJob(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		h.errorHandler.HandleError(c,
			domain.StatusBadRequest,
			"Validation error",
			[]domain.BatchError{domain.NewValidationError("job_id", "job_id is required", "")},
		)
		return
	}

	job, err := h.encryptionService.GetJobStatus(c.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			h.errorHandler.HandleError(c,
				domain.StatusNotFound,
				"Job not found",
				[]domain.BatchError{domain.NewNotFoundError("job", jobID)},
			)
			return
		}
		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to get job status",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeEncryptionFailed,
			}},
		)
		return
	}

	if err := job.CanPause(); err != nil {
		var stateErr *domain.JobStateError
		if errors.As(err, &stateErr) {
			h.errorHandler.HandleStateError(c, stateErr)
			return
		}
	}

	if err := h.encryptionService.PauseJob(c.Request.Context(), jobID); err != nil {
		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to pause job",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeEncryptionFailed,
			}},
		)
		return
	}

	c.JSON(domain.StatusOK, gin.H{
		"job_id":  jobID,
		"status":  domain.StatusPaused,
		"message": "Job paused successfully",
	})
}

// ResumeJob handles the request to resume an encryption job
func (h *EncryptionHandler) ResumeJob(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		h.errorHandler.HandleError(c,
			domain.StatusBadRequest,
			"Validation error",
			[]domain.BatchError{domain.NewValidationError("job_id", "job_id is required", "")},
		)
		return
	}

	job, err := h.encryptionService.GetJobStatus(c.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			h.errorHandler.HandleError(c,
				domain.StatusNotFound,
				"Job not found",
				[]domain.BatchError{domain.NewNotFoundError("job", jobID)},
			)
			return
		}
		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to get job status",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeEncryptionFailed,
			}},
		)
		return
	}

	if err := job.CanResume(); err != nil {
		var stateErr *domain.JobStateError
		if errors.As(err, &stateErr) {
			h.errorHandler.HandleStateError(c, stateErr)
			return
		}
	}

	if err := h.encryptionService.ResumeJob(c.Request.Context(), jobID); err != nil {
		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to resume job",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeEncryptionFailed,
			}},
		)
		return
	}

	c.JSON(domain.StatusOK, gin.H{
		"job_id":  jobID,
		"status":  domain.StatusProgress,
		"message": "Job resumed successfully",
	})
}

// StopJob handles the request to stop an encryption job
func (h *EncryptionHandler) StopJob(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		h.errorHandler.HandleError(c,
			domain.StatusBadRequest,
			"Validation error",
			[]domain.BatchError{domain.NewValidationError("job_id", "job_id is required", "")},
		)
		return
	}

	job, err := h.encryptionService.GetJobStatus(c.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			h.errorHandler.HandleError(c,
				domain.StatusNotFound,
				"Job not found",
				[]domain.BatchError{domain.NewNotFoundError("job", jobID)},
			)
			return
		}
		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to get job status",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeEncryptionFailed,
			}},
		)
		return
	}

	if err := job.CanStop(); err != nil {
		var stateErr *domain.JobStateError
		if errors.As(err, &stateErr) {
			h.errorHandler.HandleStateError(c, stateErr)
			return
		}
	}

	if err := h.encryptionService.StopJob(c.Request.Context(), jobID); err != nil {
		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to stop job",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeEncryptionFailed,
			}},
		)
		return
	}

	c.JSON(domain.StatusOK, gin.H{
		"job_id":  jobID,
		"status":  domain.StatusFailed,
		"message": "Job stopped successfully",
	})
}

// StopEngine handles the request to stop the encryption engine
func (h *EncryptionHandler) StopEngine(c *gin.Context) {
	if err := h.encryptionService.StopEngine(); err != nil {
		h.errorHandler.HandleError(c,
			domain.StatusInternalServerError,
			"Failed to stop engine",
			[]domain.BatchError{{
				Field:   "general",
				Message: err.Error(),
				Code:    domain.ErrCodeEncryptionFailed,
			}},
		)
		return
	}

	c.JSON(domain.StatusOK, gin.H{
		"message": "Engine stopped successfully",
	})
}

// Utility functions
func parseTimestamp(s string) int64 {
	if s == "" {
		return 0
	}
	// Try parsing as Unix timestamp
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ts
	}
	// Try parsing as RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix()
	}
	return 0
}

func parseFloat(s string, defaultVal float64) float64 {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return val
}