package controllers

import (
	"net/http"
	"strconv"
)

// RequestValidator handles request validation
type RequestValidator struct{}

// NewRequestValidator creates a new request validator
func NewRequestValidator() *RequestValidator {
	return &RequestValidator{}
}

// ValidateVehicleIDQuery validates vehicle_id query parameter
func (rv *RequestValidator) ValidateVehicleIDQuery(r *http.Request) (string, error) {
	vehicleID := r.URL.Query().Get("vehicle_id")
	if vehicleID == "" {
		return "", NewValidationError("vehicle_id query parameter is required")
	}
	return vehicleID, nil
}

// ValidateHoursQuery validates hours query parameter with default fallback
func (rv *RequestValidator) ValidateHoursQuery(r *http.Request) (int, error) {
	hoursStr := r.URL.Query().Get("hours")
	if hoursStr == "" {
		return 24, nil // Default to 24 hours
	}

	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours <= 0 {
		return 24, nil // Default to 24 hours for invalid input
	}

	// Limit to reasonable range (1 hour to 24 hours)
	if hours > 24 { // Max 24 hours
		hours = 24
	}

	return hours, nil
}

// ValidateVehicleDataRequest validates the vehicle data request body
func (rv *RequestValidator) ValidateVehicleDataRequest(r *http.Request) error {
	// Check content type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return NewValidationError("content-type must be application/json")
	}

	// Check if body is not empty
	if r.Body == nil {
		return NewValidationError("request body is required")
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	return ve.Message
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}
