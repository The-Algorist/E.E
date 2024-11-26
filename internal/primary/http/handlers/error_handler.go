package handlers

import (
    "github.com/gin-gonic/gin"
    "E.E/internal/core/domain"
    "E.E/internal/primary/http/middleware"
	"go.uber.org/zap"
)

// ErrorHandler provides consistent error response handling
type ErrorHandler struct {
    logger *zap.Logger
}

func NewErrorHandler(logger *zap.Logger) *ErrorHandler {
    return &ErrorHandler{logger: logger}
}

func (h *ErrorHandler) HandleError(c *gin.Context, status int, message string, errors []domain.BatchError) {
    requestID := middleware.GetRequestID(c)
    
    response := domain.NewBatchErrorResponse(
        message,
        errors,
        nil,
        requestID,
    )

    // Log error with request ID
    h.logger.Error("Request failed",
        zap.String("request_id", requestID),
        zap.Int("status", status),
        zap.String("message", message),
        zap.Any("errors", errors),
    )

    c.JSON(status, response)
}

func (h *ErrorHandler) HandleBatchError(c *gin.Context, status int, message string, errors []domain.BatchError, details *domain.BatchDetails) {
    requestID := middleware.GetRequestID(c)
    
    response := domain.NewBatchErrorResponse(
        message,
        errors,
        details,
        requestID,
    )

    // Log batch error with request ID
    h.logger.Error("Batch operation failed",
        zap.String("request_id", requestID),
        zap.Int("status", status),
        zap.String("message", message),
        zap.Any("errors", errors),
        zap.Any("details", details),
    )

    c.JSON(status, response)
}

func (h *ErrorHandler) HandleStateError(c *gin.Context, err *domain.JobStateError) {
    h.HandleBatchError(c,
        domain.StatusConflict,
        "Invalid state transition",
        []domain.BatchError{{
            Field:   "status",
            Message: err.Error(),
            Code:    domain.ErrCodeJobState,
            Value:   string(err.CurrentState),
        }},
        &domain.BatchDetails{
            Action: err.Action,
            JobIDs: []string{err.JobID},
        },
    )
}

// Add convenience methods for common error scenarios
func (h *ErrorHandler) HandleNotFound(c *gin.Context, resourceType, identifier string) {
    h.HandleError(c,
        domain.StatusNotFound,
        "Resource not found",
        []domain.BatchError{domain.NewNotFoundError(resourceType, identifier)},
    )
}

func (h *ErrorHandler) HandleValidationError(c *gin.Context, field, message string) {
    h.HandleError(c,
        domain.StatusBadRequest,
        "Validation error",
        []domain.BatchError{domain.NewValidationError(field, message, "")},
    )
}

func (h *ErrorHandler) HandleInternalError(c *gin.Context, err error) {
    h.HandleError(c,
        domain.StatusInternalServerError,
        "Internal server error",
        []domain.BatchError{{
            Field:   "general",
            Message: err.Error(),
            Code:    domain.ErrCodeEncryptionFailed,
        }},
    )
}