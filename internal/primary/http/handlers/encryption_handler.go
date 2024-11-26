package handlers
import (
	"net/http"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
	"time"
	"strings"
	"errors"

	"E.E/internal/core/domain"
	"E.E/internal/core/ports"
	"E.E/internal/core/services"
	
)

type EncryptionHandler struct {
	encryptionService ports.EncryptionService
	logger           *zap.Logger
}

func NewEncryptionHandler(service ports.EncryptionService, logger *zap.Logger) *EncryptionHandler {
	return &EncryptionHandler{
		encryptionService: service,
		logger:           logger,

	}
}

// StartEncryption handles the request to start video encryption
func (h *EncryptionHandler) StartEncryption(c *gin.Context) {
	var req domain.EncryptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request format", zap.Error(err))
		c.JSON(http.StatusBadRequest, domain.NewBatchErrorResponse(
			"Invalid request format",
			[]domain.BatchError{{
				Field:   "request",
				Message: err.Error(),
				Code:    domain.ErrCodeInvalidFormat,
			}},
			nil,
		))
		return
	}

	if req.Batch {
		op := domain.BatchOperation{
			Action:     req.Action,
			SourceURLs: req.SourceURLs,
			JobIDs:     req.JobIDs,
		}
		
		result, err := h.encryptionService.ProcessBatch(c.Request.Context(), op)
		if err != nil {
			h.logger.Error("Failed to process batch", zap.Error(err))
			
			var jobStateErr *domain.JobStateError
			if errors.As(err, &jobStateErr) {
				c.JSON(http.StatusConflict, domain.NewBatchErrorResponse(
					"Invalid job state",
					[]domain.BatchError{domain.ConvertJobStateErrorToBatchError(jobStateErr)},
					&domain.BatchDetails{
						Action: string(op.Action),
						JobIDs: op.JobIDs,
					},
				))
				return
			}

			c.JSON(http.StatusBadRequest, domain.NewBatchErrorResponse(
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
			))
			return
		}
		
		c.JSON(http.StatusAccepted, result)
		return
	}

	// Handle single file operation
	if req.SourceURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "source_url is required for single operations",
		})
		return
	}

	job, err := h.encryptionService.StartEncryption(c.Request.Context(), req.SourceURL)
	if err != nil {
		h.logger.Error("Failed to start encryption", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start encryption",
		})
		return
	}

	response := domain.EncryptionResponse{
		JobID:     job.ID,
		Status:    job.Status,
		CreatedAt: job.CreatedAt,
	}

	c.JSON(http.StatusAccepted, response)
}

// GetStatus handles the request to check encryption status
func (h *EncryptionHandler) GetStatus(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		errResp := domain.NewBatchErrorResponse(
			"Validation error",
			[]domain.BatchError{domain.NewValidationError("job_id", "job_id is required", "")},
			nil,
		)
		c.JSON(domain.StatusBadRequest, errResp)
		return
	}

	job, err := h.encryptionService.GetJobStatus(c.Request.Context(), jobID)
	if err != nil {
		h.logger.Error("Failed to get job status", 
			zap.Error(err), 
			zap.String("jobId", jobID),
		)
		
		if errors.Is(err, domain.ErrJobNotFound) {
			errResp := domain.NewBatchErrorResponse(
				"Job not found",
				[]domain.BatchError{domain.NewNotFoundError("job", jobID)},
				nil,
			)
			c.JSON(domain.StatusNotFound, errResp)
			return
		}

		errResp, status := domain.GetBatchErrorResponse(err, "get_status")
		c.JSON(status, errResp)
		return
	}

	c.JSON(domain.StatusOK, job)
}

// PauseJob handles the request to pause an encryption job
func (h *EncryptionHandler) PauseJob(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		errResp := domain.NewBatchErrorResponse(
			"Validation error",
			[]domain.BatchError{domain.NewValidationError("job_id", "job_id is required", "")},
			nil,
		)
		c.JSON(domain.StatusBadRequest, errResp)
		return
	}

	if err := h.encryptionService.PauseJob(c.Request.Context(), jobID); err != nil {
		h.logger.Error("Failed to pause job", 
			zap.Error(err), 
			zap.String("jobId", jobID),
		)

		var jobStateErr *domain.JobStateError
		if errors.As(err, &jobStateErr) {
			errResp := domain.NewBatchErrorResponse(
				"Invalid job state",
				[]domain.BatchError{domain.ConvertJobStateErrorToBatchError(jobStateErr)},
				&domain.BatchDetails{
					Action: "pause",
					JobIDs: []string{jobID},
				},
			)
			c.JSON(domain.StatusConflict, errResp)
			return
		}

		errResp, status := domain.GetBatchErrorResponse(err, "pause")
		c.JSON(status, errResp)
		return
	}

	c.JSON(domain.StatusOK, gin.H{
		"message": "Job paused successfully",
		"job_id":  jobID,
		"status":  domain.StatusPaused,
	})
}

// ResumeJob handles the request to resume an encryption job
func (h *EncryptionHandler) ResumeJob(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		errResp := domain.NewBatchErrorResponse(
			"Validation error",
			[]domain.BatchError{domain.NewValidationError("job_id", "job_id is required", "")},
			nil,
		)
		c.JSON(domain.StatusBadRequest, errResp)
		return
	}

	if err := h.encryptionService.ResumeJob(c.Request.Context(), jobID); err != nil {
		h.logger.Error("Failed to resume job", 
			zap.Error(err), 
			zap.String("jobId", jobID),
		)

		if errors.Is(err, domain.ErrJobNotFound) {
			errResp := domain.NewBatchErrorResponse(
				"Job not found",
				[]domain.BatchError{domain.NewNotFoundError("job", jobID)},
				nil,
			)
			c.JSON(domain.StatusNotFound, errResp)
			return
		}

		errResp, status := domain.GetBatchErrorResponse(err, "resume")
		c.JSON(status, errResp)
		return
	}

	c.JSON(domain.StatusOK, gin.H{
		"message": "Job resumed successfully",
		"job_id":  jobID,
		"status":  domain.StatusProgress,
	})
}

// StopEngine handles the request to stop the encryption engine
func (h *EncryptionHandler) StopEngine(c *gin.Context) {
	if err := h.encryptionService.StopEngine(); err != nil {
		h.logger.Error("Failed to stop encryption engine", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to stop encryption engine",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Encryption engine stopped successfully",
	})
}

// StopJob handles the request to stop a specific encryption job
func (h *EncryptionHandler) StopJob(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "job_id is required",
		})
		return
	}

	if err := h.encryptionService.StopJob(c.Request.Context(), jobID); err != nil {
		h.logger.Error("Failed to stop job", 
			zap.Error(err), 
			zap.String("jobId", jobID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to stop job",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job stopped successfully",
		"job_id":  jobID,
		"status":  domain.StatusFailed,
	})
}

// ListJobs returns the list of all jobs with their history
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
			)
			c.JSON(domain.StatusBadRequest, errResp)
			return
		}
		
		h.logger.Error("Failed to list jobs", zap.Error(err))
		errResp, status := domain.GetBatchErrorResponse(err, "list")
		c.JSON(status, errResp)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  len(jobs),
		},
		"filter": filter,
		"sort": sort,
		"available_sort_options": services.GetAvailableSortOptions(),
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

	c.JSON(http.StatusOK, summary)
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

func parseTimestamp(s string) int64 {
	// Try parsing as Unix timestamp first
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ts
	}

	// Try parsing as RFC3339 date
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix()
	}

	// Try parsing as simple date (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Unix()
	}

	return 0
}

// ProcessBatch handles the request to process a batch of encryption jobs
func (h *EncryptionHandler) ProcessBatch(c *gin.Context) {
	var op domain.BatchOperation
	if err := c.ShouldBindJSON(&op); err != nil {
		h.logger.Error("Invalid batch operation request", zap.Error(err))
		errResp := domain.NewBatchErrorResponse(
			"Invalid request format",
			[]domain.BatchError{{
				Field:   "request",
				Message: err.Error(),
				Code:    domain.ErrCodeInvalidFormat,
			}},
			nil,
		)
		c.JSON(domain.StatusBadRequest, errResp)
		return
	}

	result, err := h.encryptionService.ProcessBatch(c.Request.Context(), op)
	if err != nil {
		h.logger.Error("Failed to process batch operation", zap.Error(err))
		errResp, status := domain.GetBatchErrorResponse(err, string(op.Action))
		c.JSON(status, errResp)
		return
	}

	c.JSON(domain.StatusOK, result)
}

// GetBatchResult handles the request to retrieve a batch operation result
func (h *EncryptionHandler) GetBatchResult(c *gin.Context) {
	batchID := c.Param("batchId")
	if batchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch_id is required"})
		return
	}

	result, err := h.encryptionService.GetBatchResult(c.Request.Context(), batchID)
	if err != nil {
		h.logger.Error("Failed to get batch result", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get batch result"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetJobHistory handles the request to retrieve job history
func (h *EncryptionHandler) GetJobHistory(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	history, err := h.encryptionService.GetJobHistory(c.Request.Context(), jobID)
	if err != nil {
		h.logger.Error("Failed to get job history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job history"})
		return
	}

	c.JSON(http.StatusOK, history)
}