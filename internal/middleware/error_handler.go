package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"fleet-tracking/pkg/logger"
)

// StandardErrorResponse represents a standardized error response
type StandardErrorResponse struct {
	Error     string            `json:"error"`
	Code      string            `json:"code,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
	Timestamp string            `json:"timestamp"`
}

// ErrorHandler provides centralized error handling with proper HTTP status codes
type ErrorHandler struct{}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// WriteError writes a standardized error response
func (eh *ErrorHandler) WriteError(w http.ResponseWriter, r *http.Request, statusCode int, message string, details map[string]string) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Create error response
	errorResp := StandardErrorResponse{
		Error:     message,
		Details:   details,
		Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Add request ID if available
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		errorResp.RequestID = requestID
	}

	// Log the error with context
	eh.logError(r, statusCode, message, details)

	// Write JSON response
	json.NewEncoder(w).Encode(errorResp)
}

// WriteValidationError writes a validation error response
func (eh *ErrorHandler) WriteValidationError(w http.ResponseWriter, r *http.Request, message string) {
	eh.WriteError(w, r, http.StatusBadRequest, message, map[string]string{
		"type": "validation_error",
	})
}

// WriteNotFoundError writes a not found error response
func (eh *ErrorHandler) WriteNotFoundError(w http.ResponseWriter, r *http.Request, message string) {
	eh.WriteError(w, r, http.StatusNotFound, message, map[string]string{
		"type": "not_found",
	})
}

// WriteUnauthorizedError writes an unauthorized error response
func (eh *ErrorHandler) WriteUnauthorizedError(w http.ResponseWriter, r *http.Request, message string) {
	eh.WriteError(w, r, http.StatusUnauthorized, message, map[string]string{
		"type": "unauthorized",
	})
}

// WriteForbiddenError writes a forbidden error response
func (eh *ErrorHandler) WriteForbiddenError(w http.ResponseWriter, r *http.Request, message string) {
	eh.WriteError(w, r, http.StatusForbidden, message, map[string]string{
		"type": "forbidden",
	})
}

// WriteInternalServerError writes an internal server error response
func (eh *ErrorHandler) WriteInternalServerError(w http.ResponseWriter, r *http.Request, message string) {
	eh.WriteError(w, r, http.StatusInternalServerError, message, map[string]string{
		"type": "internal_server_error",
	})
}

// WriteServiceUnavailableError writes a service unavailable error response
func (eh *ErrorHandler) WriteServiceUnavailableError(w http.ResponseWriter, r *http.Request, message string) {
	eh.WriteError(w, r, http.StatusServiceUnavailable, message, map[string]string{
		"type": "service_unavailable",
	})
}

// WriteDatabaseError writes a database error response
func (eh *ErrorHandler) WriteDatabaseError(w http.ResponseWriter, r *http.Request, message string) {
	eh.WriteError(w, r, http.StatusInternalServerError, message, map[string]string{
		"type": "database_error",
	})
}

// WriteCacheError writes a cache error response
func (eh *ErrorHandler) WriteCacheError(w http.ResponseWriter, r *http.Request, message string) {
	eh.WriteError(w, r, http.StatusInternalServerError, message, map[string]string{
		"type": "cache_error",
	})
}

// logError logs the error with comprehensive context
func (eh *ErrorHandler) logError(r *http.Request, statusCode int, message string, details map[string]string) {
	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	caller := "unknown"
	if ok {
		caller = fmt.Sprintf("%s:%d", file, line)
	}

	// Log error with context
	logger.Error(fmt.Sprintf("HTTP_ERROR: %s %s - Status: %d - Message: %s - Caller: %s - Details: %v",
		r.Method, r.URL.Path, statusCode, message, caller, details))

	// Log additional context for debugging
	logger.Error(fmt.Sprintf("ERROR_CONTEXT: RemoteAddr=%s UserAgent=%s ContentLength=%d",
		r.RemoteAddr, r.UserAgent(), r.ContentLength))

	// Log query parameters if present
	if len(r.URL.Query()) > 0 {
		logger.Error(fmt.Sprintf("ERROR_QUERY_PARAMS: %v", r.URL.Query()))
	}
}

// RecoverMiddleware provides panic recovery with proper error handling
func RecoverMiddleware(errorHandler *ErrorHandler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic
					logger.Error(fmt.Sprintf("PANIC_RECOVERED: %s %s - Error: %v",
						r.Method, r.URL.Path, err))

					// Get stack trace
					stack := make([]byte, 4096)
					length := runtime.Stack(stack, false)
					logger.Error(fmt.Sprintf("STACK_TRACE: %s", string(stack[:length])))

					// Write error response
					errorHandler.WriteInternalServerError(w, r, "Internal server error occurred")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
