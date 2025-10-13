package middleware

import (
	"encoding/json"
	"fleet-tracking/internal/auth"
	"fleet-tracking/pkg/logger"
	"fmt"
	"net/http"
	"time"
)

// Logger middleware for request logging
func Logger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			logger.Info(fmt.Sprintf("Request: %s %s - %d - %v", r.Method, r.URL.Path, wrapped.statusCode, duration))
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// writeErrorResponse writes an error response in JSON format
func writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   errorMsg,
		Message: message,
		Code:    statusCode,
	}

	json.NewEncoder(w).Encode(response)
}

// JWT middleware for authentication
func JWT() func(http.Handler) http.Handler {
	jwtManager := auth.NewJWTManager()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			tokenString, err := auth.ExtractTokenFromHeader(authHeader)
			if err != nil {
				logger.Error("JWT middleware: failed to extract token: ", err)
				writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authorization header is required")
				return
			}

			// Validate token
			_, err = jwtManager.ValidateToken(tokenString)
			if err != nil {
				logger.Error("JWT middleware: failed to validate token: ", err)
				writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
				return
			}

			// Add authentication status to context
			ctx := auth.WithAuthentication(r.Context(), true)
			r = r.WithContext(ctx)

			logger.Info("JWT middleware: request authenticated")

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// Middleware Chaining (Concept: https://github.com/justinas/alice/blob/master/chain.go)
func Chain(handler http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) http.HandlerFunc {
	h := http.Handler(handler)
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h.ServeHTTP
}
