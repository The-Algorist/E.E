package handlers

import (
    "net/http"
    "fmt"
    "strings"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "E.E/internal/core/domain"
    "E.E/internal/core/services"
)

type BatchHandler struct {
    batchService *services.BatchService
    logger       *zap.Logger
}

func NewBatchHandler(batchService *services.BatchService, logger *zap.Logger) *BatchHandler {
    return &BatchHandler{
        batchService: batchService,
        logger:       logger,
    }
}

func (h *BatchHandler) ProcessBatch(c *gin.Context) {
    var op domain.BatchOperation
    if err := c.ShouldBindJSON(&op); err != nil {
        h.logger.Error("Invalid batch operation request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
        return
    }

    if len(op.JobIDs) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No job IDs provided"})
        return
    }

    result, err := h.batchService.ProcessBatch(c.Request.Context(), op)
    if err != nil {
        h.logger.Error("Failed to process batch operation", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process batch operation"})
        return
    }

    c.JSON(http.StatusOK, result)
}

func (h *BatchHandler) GetBatchOperation(c *gin.Context) {
    batchID := c.Param("batchId")
    if batchID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "batch ID is required"})
        return
    }

    result, err := h.batchService.GetBatchResult(c.Request.Context(), batchID)
    if err != nil {
        if strings.Contains(err.Error(), "not found") {
            c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("batch operation %s not found", batchID)})
            return
        }
        h.logger.Error("Failed to get batch operation",
            zap.String("batch_id", batchID),
            zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get batch operation"})
        return
    }

    c.JSON(http.StatusOK, result)
}

func (h *BatchHandler) ListBatchResults(c *gin.Context) {
    filter := domain.BatchFilter{
        Status: c.Query("status"),
    }
    
    if jobIDs := c.Query("job_ids"); jobIDs != "" {
        filter.JobIDs = strings.Split(jobIDs, ",")
    }

    results, err := h.batchService.ListBatchResults(c.Request.Context(), filter)
    if err != nil {
        h.logger.Error("Failed to list batch results", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list batch results"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "results": results,
    })
}