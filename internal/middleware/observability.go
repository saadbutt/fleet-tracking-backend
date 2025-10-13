package middleware

import (
	"fmt"
	"net/http"
	"time"

	"fleet-tracking/pkg/logger"
)

// ObservabilityMiddleware provides comprehensive request logging and observability
func ObservabilityMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the response writer to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default status
			}

			// Log request start
			logger.Info(fmt.Sprintf("Request started: %s %s from %s",
				r.Method, r.URL.Path, r.RemoteAddr))

			// Process the request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Log request completion with comprehensive details
			logger.Info(fmt.Sprintf("Request completed: %s %s - Status: %d - Duration: %v - Size: %d bytes - User-Agent: %s",
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				duration,
				r.ContentLength,
				r.UserAgent()))

			// Log slow requests (>1 second)
			if duration > time.Second {
				logger.Info(fmt.Sprintf("SLOW REQUEST: %s %s took %v",
					r.Method, r.URL.Path, duration))
			}

			// Log error responses
			if wrapped.statusCode >= 400 {
				logger.Error(fmt.Sprintf("Error response: %s %s - Status: %d - Duration: %v",
					r.Method, r.URL.Path, wrapped.statusCode, duration))
			}
		})
	}
}

// RequestMetrics captures request metrics for observability
type RequestMetrics struct {
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
	Size       int64
	UserAgent  string
	RemoteAddr string
	Timestamp  time.Time
}

// GetRequestMetrics extracts metrics from the request
func GetRequestMetrics(r *http.Request, statusCode int, duration time.Duration, size int64) RequestMetrics {
	return RequestMetrics{
		Method:     r.Method,
		Path:       r.URL.Path,
		StatusCode: statusCode,
		Duration:   duration,
		Size:       size,
		UserAgent:  r.UserAgent(),
		RemoteAddr: r.RemoteAddr,
		Timestamp:  time.Now(),
	}
}

// LogRequestMetrics logs structured request metrics
func LogRequestMetrics(metrics RequestMetrics) {
	// Log basic request info
	logger.Info(fmt.Sprintf("REQUEST_METRICS: method=%s path=%s status=%d duration=%v size=%d",
		metrics.Method, metrics.Path, metrics.StatusCode, metrics.Duration, metrics.Size))

	// Log performance metrics
	if metrics.Duration > 500*time.Millisecond {
		logger.Info(fmt.Sprintf("PERFORMANCE_WARNING: %s %s took %v",
			metrics.Method, metrics.Path, metrics.Duration))
	}

	// Log error metrics
	if metrics.StatusCode >= 400 {
		logger.Error(fmt.Sprintf("ERROR_METRICS: %s %s returned %d",
			metrics.Method, metrics.Path, metrics.StatusCode))
	}
}
