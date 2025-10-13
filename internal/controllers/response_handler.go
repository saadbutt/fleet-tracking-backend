package controllers

import (
	"encoding/json"
	"net/http"
)

// ResponseHandler handles HTTP responses with consistent formatting
type ResponseHandler struct{}

// NewResponseHandler creates a new response handler
func NewResponseHandler() *ResponseHandler {
	return &ResponseHandler{}
}

// WriteJSONResponse writes a JSON response with the given status code
func (rh *ResponseHandler) WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// WriteErrorResponse writes an error response with consistent format
func (rh *ResponseHandler) WriteErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	errorResponse := map[string]string{
		"error": message,
	}
	rh.WriteJSONResponse(w, statusCode, errorResponse)
}

// WriteSuccessResponse writes a success response with consistent format
func (rh *ResponseHandler) WriteSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	rh.WriteJSONResponse(w, statusCode, data)
}

// WriteValidationError writes a validation error response
func (rh *ResponseHandler) WriteValidationError(w http.ResponseWriter, message string) {
	rh.WriteErrorResponse(w, http.StatusBadRequest, message)
}

// WriteInternalServerError writes an internal server error response
func (rh *ResponseHandler) WriteInternalServerError(w http.ResponseWriter, message string) {
	rh.WriteErrorResponse(w, http.StatusInternalServerError, message)
}

// WriteNotFoundError writes a not found error response
func (rh *ResponseHandler) WriteNotFoundError(w http.ResponseWriter, message string) {
	rh.WriteErrorResponse(w, http.StatusNotFound, message)
}

// WriteUnauthorizedError writes an unauthorized error response
func (rh *ResponseHandler) WriteUnauthorizedError(w http.ResponseWriter, message string) {
	rh.WriteErrorResponse(w, http.StatusUnauthorized, message)
}
