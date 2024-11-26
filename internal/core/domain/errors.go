package domain

import (
    "fmt"
    "net/http"

)

// JobStateError represents an error with job state transition
type JobStateError struct {
    JobID        string
    CurrentState EncryptionStatus
    Action       string
    Message      string
}

func (e *JobStateError) Error() string {
    return fmt.Sprintf("invalid job state transition: cannot %s job %s (current state: %s) - %s",
        e.Action, e.JobID, e.CurrentState, e.Message)
}

// NewJobStateError creates a new JobStateError
func NewJobStateError(jobID string, currentState EncryptionStatus, action, message string) error {
    return &JobStateError{
        JobID:        jobID,
        CurrentState: currentState,
        Action:       action,
        Message:      message,
    }
}

// IsJobStateError checks if an error is a JobStateError
func IsJobStateError(err error) bool {
    _, ok := err.(*JobStateError)
    return ok
}

// BatchErrorResponse represents a structured error response
type BatchErrorResponse struct {
    Status  string         `json:"status"`
    Message string         `json:"message"`
    Errors  []BatchError   `json:"errors,omitempty"`
    Details *BatchDetails  `json:"details,omitempty"`
    RequestID string        `json:"request_id,omitempty"`
}

type BatchError struct {
    Field      string `json:"field"`
    Message    string `json:"message"`
    Value      string `json:"value,omitempty"`
    Code       string `json:"code"`
    ActionType string `json:"action_type,omitempty"`
}

type BatchDetails struct {
    BatchID    string   `json:"batch_id,omitempty"`
    Action     string   `json:"action,omitempty"`
    JobIDs     []string `json:"job_ids,omitempty"`
    SourceURLs []string `json:"source_urls,omitempty"`
}

// Error codes
const (
    ErrCodeInvalidFormat   = "invalid_format"
    ErrCodeValidation      = "validation_error"
    ErrCodeJobState        = "job_state_error"
    ErrCodeNotFound        = "not_found"
    ErrCodeBatchOperation  = "batch_operation_error"
    ErrCodeUnauthorized    = "unauthorized"
    ErrCodeForbidden       = "forbidden"
    ErrCodeTimeout         = "timeout"
    ErrCodeRateLimit       = "rate_limit"
    ErrCodeJobNotFound     = "job_not_found"
    ErrCodeBatchNotFound   = "batch_not_found"
    ErrCodeInvalidState    = "invalid_state"
    ErrCodeInvalidAction   = "invalid_action"
    ErrCodeEncryptionFailed = "encryption_failed"
)

// HTTP Status codes
const (
    StatusOK                  = http.StatusOK
    StatusAccepted           = http.StatusAccepted
    StatusBadRequest         = http.StatusBadRequest
    StatusUnauthorized       = http.StatusUnauthorized
    StatusForbidden          = http.StatusForbidden
    StatusNotFound           = http.StatusNotFound
    StatusConflict           = http.StatusConflict
    StatusTooManyRequests    = http.StatusTooManyRequests
    StatusInternalServerError = http.StatusInternalServerError
    StatusServiceUnavailable = http.StatusServiceUnavailable
    StatusGatewayTimeout     = http.StatusGatewayTimeout
)

// Extended ErrorStatusMap
var ErrorStatusMap = map[string]int{
    ErrCodeInvalidFormat:    StatusBadRequest,
    ErrCodeValidation:       StatusBadRequest,
    ErrCodeJobState:         StatusConflict,
    ErrCodeNotFound:         StatusNotFound,
    ErrCodeBatchOperation:   StatusBadRequest,
    ErrCodeUnauthorized:     StatusUnauthorized,
    ErrCodeForbidden:        StatusForbidden,
    ErrCodeTimeout:          StatusGatewayTimeout,
    ErrCodeRateLimit:        StatusTooManyRequests,
    ErrCodeJobNotFound:      StatusNotFound,
    ErrCodeBatchNotFound:    StatusNotFound,
    ErrCodeInvalidState:     StatusConflict,
    ErrCodeInvalidAction:    StatusBadRequest,
    ErrCodeEncryptionFailed: StatusInternalServerError,
}

// NewBatchErrorResponse creates a new BatchErrorResponse
func NewBatchErrorResponse(message string, errs []BatchError, details *BatchDetails, requestID string) BatchErrorResponse {
    return BatchErrorResponse{
        Status:    "error",
        Message:   message,
        Errors:    errs,
        Details:   details,
        RequestID: requestID,
    }
}

// ConvertJobStateErrorToBatchError converts a JobStateError to a BatchError
func ConvertJobStateErrorToBatchError(err *JobStateError) BatchError {
    return BatchError{
        Field:      "job_state",
        Message:    err.Error(),
        Value:      string(err.CurrentState),
        Code:       ErrCodeJobState,
        ActionType: err.Action,
    }
}

// BatchValidationError represents a validation error for batch operations
type BatchValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Value   string `json:"value,omitempty"`
}

// ToBatchError converts a BatchValidationError to a BatchError
func (e BatchValidationError) ToBatchError(action string) BatchError {
    return BatchError{
        Field:      e.Field,
        Message:    e.Message,
        Value:      e.Value,
        Code:       ErrCodeValidation,
        ActionType: action,
    }
}

// NewValidationError creates a BatchError for validation errors
func NewValidationError(field, message string, value string) BatchError {
    return BatchError{
        Field:   field,
        Message: message,
        Value:   value,
        Code:    ErrCodeValidation,
    }
}

// NewNotFoundError creates a BatchError for not found errors
func NewNotFoundError(resourceType, identifier string) BatchError {
    return BatchError{
        Field:   resourceType,
        Message: fmt.Sprintf("%s not found: %s", resourceType, identifier),
        Value:   identifier,
        Code:    ErrCodeNotFound,
    }
}

// NewEncryptionError creates a BatchError for encryption-related errors
func NewEncryptionError(message string, details string) BatchError {
    return BatchError{
        Field:   "encryption",
        Message: message,
        Value:   details,
        Code:    ErrCodeEncryptionFailed,
    }
}

// GetBatchErrorResponse creates a complete error response with appropriate status
func GetBatchErrorResponse(err error, action string) (BatchErrorResponse, int) {
    var batchError BatchError
    var details *BatchDetails

    switch e := err.(type) {
    case *JobStateError:
        batchError = ConvertJobStateErrorToBatchError(e)
        details = &BatchDetails{Action: action}
    case *BatchValidationError:
        batchError = e.ToBatchError(action)
    case *BatchError:
        batchError = *e
    default:
        batchError = BatchError{
            Field:      "general",
            Message:    err.Error(),
            Code:       ErrCodeBatchOperation,
            ActionType: action,
        }
    }

    response := NewBatchErrorResponse(
        "Operation failed",
        []BatchError{batchError},
        details,
        "",
    )

    return response, GetBatchErrorHTTPStatus(response)
}

// IsRetryableError determines if the error can be retried
func IsRetryableError(code string) bool {
    status, ok := ErrorStatusMap[code]
    if !ok {
        return false
    }
    return status == StatusServiceUnavailable || 
           status == StatusGatewayTimeout || 
           status == StatusTooManyRequests
}

// IsClientError checks if the error code represents a client error (4xx)
func IsClientError(code string) bool {
    status, ok := ErrorStatusMap[code]
    return ok && status >= 400 && status < 500
}

// IsServerError checks if the error code represents a server error (5xx)
func IsServerError(code string) bool {
    status, ok := ErrorStatusMap[code]
    return ok && status >= 500
}

// GetErrorHTTPStatus returns the appropriate HTTP status code for a BatchError
func GetErrorHTTPStatus(err BatchError) int {
    if status, ok := ErrorStatusMap[err.Code]; ok {
        return status
    }
    return StatusInternalServerError
}

// GetBatchErrorHTTPStatus returns the appropriate HTTP status code for a BatchErrorResponse
func GetBatchErrorHTTPStatus(response BatchErrorResponse) int {
    if len(response.Errors) == 0 {
        return StatusInternalServerError
    }
    
    // Return the highest status code from all errors
    maxStatus := StatusBadRequest
    for _, err := range response.Errors {
        status := GetErrorHTTPStatus(err)
        if status > maxStatus {
            maxStatus = status
        }
    }
    return maxStatus
}

// GetHTTPStatusForError returns the appropriate HTTP status code for a generic error
func GetHTTPStatusForError(err error) int {
    switch e := err.(type) {
    case *JobStateError:
        return StatusConflict
    case *BatchValidationError:
        return StatusBadRequest
    case *BatchError:
        return GetErrorHTTPStatus(*e)
    default:
        return StatusInternalServerError
    }
}

// Implement error interface for BatchValidationError
func (e *BatchValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Implement error interface for BatchError
func (e BatchError) Error() string {
    if e.Value != "" {
        return fmt.Sprintf("%s: %s (value: %s)", e.Field, e.Message, e.Value)
    }
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Add these with the other error definitions
var (
    // Common errors
    ErrJobNotFound = fmt.Errorf("job not found")
    ErrBatchNotFound = fmt.Errorf("batch not found")
    ErrInvalidJobState = fmt.Errorf("invalid job state")
)