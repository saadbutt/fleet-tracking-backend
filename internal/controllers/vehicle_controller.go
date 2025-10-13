package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"fleet-tracking/internal/middleware"
	"fleet-tracking/internal/model"
	"fleet-tracking/internal/service"
	"fleet-tracking/pkg/logger"
)

// HealthController handles health check endpoints
type HealthController struct {
	healthService service.HealthService
}

// VehicleController handles vehicle endpoints
type VehicleController struct {
	vehicleService   service.VehicleService
	responseHandler  *ResponseHandler
	requestValidator *RequestValidator
	errorHandler     *middleware.ErrorHandler
}

// NewHealthController creates a new health controller
func NewHealthController(healthService service.HealthService) *HealthController {
	return &HealthController{
		healthService: healthService,
	}
}

// NewVehicleController creates a new vehicle controller
func NewVehicleController(vehicleService service.VehicleService) *VehicleController {
	return &VehicleController{
		vehicleService:   vehicleService,
		responseHandler:  NewResponseHandler(),
		requestValidator: NewRequestValidator(),
		errorHandler:     middleware.NewErrorHandler(),
	}
}

// HealthCheck handles GET /health endpoint
// @Summary Health check
// @Description Get the health status of the API
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} model.HealthResponse
// @Failure 500 {object} map[string]string
// @Router /health [get]
func (hc *HealthController) HealthCheck(w http.ResponseWriter, r *http.Request) {
	logger.Info("Health check endpoint called")

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Call service layer
	health, err := hc.healthService.CheckHealth()
	if err != nil {
		logger.Error("Health check failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Health check failed",
		})
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// GetVehicleStatus handles GET /api/vehicle/status
// @Summary Get vehicle status
// @Description Get the current status of a vehicle by ID
// @Tags vehicle
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param vehicle_id query string true "Vehicle ID"
// @Success 200 {object} model.VehicleStatus
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/vehicle/status [get]
func (vc *VehicleController) GetVehicleStatus(w http.ResponseWriter, r *http.Request) {
	logger.Info("Vehicle status endpoint called")

	// Validate request parameters
	vehicleID, err := vc.requestValidator.ValidateVehicleIDQuery(r)
	if err != nil {
		logger.Error("Validation error: ", err)
		vc.errorHandler.WriteValidationError(w, r, err.Error())
		return
	}

	// Get vehicle status from service
	status, err := vc.vehicleService.GetVehicleStatus(vehicleID)
	if err != nil {
		logger.Error("Service error: ", err)
		if err.Error() == "vehicle not found" {
			vc.errorHandler.WriteNotFoundError(w, r, "Vehicle not found")
		} else {
			vc.errorHandler.WriteInternalServerError(w, r, "Failed to get vehicle status")
		}
		return
	}

	// Return successful response
	vc.responseHandler.WriteSuccessResponse(w, http.StatusOK, status)
}

// GetTripHistory handles GET /api/vehicle/trips
// @Summary Get trip history
// @Description Get trip history for a vehicle in the past 1-24 hours
// @Tags vehicle
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param vehicle_id query string true "Vehicle ID"
// @Param hours query int false "Number of hours to look back (default: 24, max: 24)"
// @Success 200 {object} model.TripHistoryResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/vehicle/trips [get]
func (vc *VehicleController) GetTripHistory(w http.ResponseWriter, r *http.Request) {
	logger.Info("Trip history endpoint called")

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Validate request parameters
	vehicleID, err := vc.requestValidator.ValidateVehicleIDQuery(r)
	if err != nil {
		logger.Error("Validation error: ", err)
		vc.errorHandler.WriteValidationError(w, r, err.Error())
		return
	}

	hours, err := vc.requestValidator.ValidateHoursQuery(r)
	if err != nil {
		logger.Error("Validation error: ", err)
		vc.errorHandler.WriteValidationError(w, r, err.Error())
		return
	}

	// Call service layer
	history, err := vc.vehicleService.GetTripHistory(vehicleID, hours)
	if err != nil {
		logger.Error("Failed to get trip history: ", err)
		if err.Error() == "vehicle not found" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(history)
}

// IngestVehicleData handles POST /api/vehicle/ingest
// @Summary Ingest vehicle data
// @Description Ingest new vehicle data (used by simulated stream). Only accepts the simulated input format with vehicle_id and status object containing timestamp.
// @Tags vehicle
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body object true "Vehicle data with vehicle_id and status object containing timestamp"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/vehicle/ingest [post]
func (vc *VehicleController) IngestVehicleData(w http.ResponseWriter, r *http.Request) {
	logger.Info("Vehicle data ingest endpoint called")

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Validate basic request structure
	if err := vc.requestValidator.ValidateVehicleDataRequest(r); err != nil {
		logger.Error("Validation error: ", err)
		vc.errorHandler.WriteValidationError(w, r, err.Error())
		return
	}

	// Parse request body
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		logger.Error("Failed to parse request body: ", err)
		vc.errorHandler.WriteValidationError(w, r, "Invalid JSON format")
		return
	}

	// Extract vehicle_id (required)
	vehicleID := getStringFromMap(data, "vehicle_id")
	if vehicleID == "" {
		vc.errorHandler.WriteValidationError(w, r, "vehicle_id is required")
		return
	}

	// Extract status object (required)
	status := getMapFromMap(data, "status")
	if len(status) == 0 {
		vc.errorHandler.WriteValidationError(w, r, "status object is required")
		return
	}

	// Validate timestamp in status object (required)
	if statusTimestamp, ok := status["timestamp"]; ok {
		if timestampStr, ok := statusTimestamp.(string); ok {
			// Validate timestamp format but keep it in status
			if _, err := time.Parse(time.RFC3339, timestampStr); err != nil {
				vc.errorHandler.WriteValidationError(w, r, "invalid timestamp format, expected Format in string 2025-06-17T09:12:00Z")
				return
			}
		} else {
			vc.errorHandler.WriteValidationError(w, r, "timestamp must be a string")
			return
		}
	} else {
		vc.errorHandler.WriteValidationError(w, r, "timestamp is required in status object")
		return
	}

	// Convert to VehicleData struct
	vehicleData := &model.VehicleData{
		VehicleID: vehicleID,
		Status:    status,
	}

	// Call service layer
	err := vc.vehicleService.IngestVehicleData(vehicleData)
	if err != nil {
		logger.Error("Failed to ingest vehicle data: ", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Return success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Vehicle data ingested successfully",
	})
}

// Helper functions for parsing JSON data
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getMapFromMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key]; ok {
		if mapVal, ok := val.(map[string]interface{}); ok {
			return mapVal
		}
	}
	return make(map[string]interface{})
}
