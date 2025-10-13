package service

import (
	"context"
	"errors"
	"math"
	"time"

	"fleet-tracking/internal/model"
	"fleet-tracking/internal/repository"
	"fleet-tracking/pkg/logger"
)

// HealthService interface for health check business logic
type HealthService interface {
	CheckHealth() (*model.HealthResponse, error)
}

// VehicleService interface for vehicle business logic
type VehicleService interface {
	GetVehicleStatus(vehicleID string) (*model.VehicleStatus, error)
	GetTripHistory(vehicleID string, hours int) (*model.TripHistoryResponse, error)
	IngestVehicleData(data *model.VehicleData) error
}

// healthService implements HealthService
type healthService struct {
	healthRepo repository.HealthRepository
}

// vehicleService implements VehicleService
type vehicleService struct {
	vehicleRepo  repository.VehicleRepository
	cacheService CacheService
}

// NewHealthService creates a new health service
func NewHealthService(healthRepo repository.HealthRepository) HealthService {
	return &healthService{
		healthRepo: healthRepo,
	}
}

// NewVehicleService creates a new vehicle service
func NewVehicleService(vehicleRepo repository.VehicleRepository, cacheService CacheService) VehicleService {
	return &vehicleService{
		vehicleRepo:  vehicleRepo,
		cacheService: cacheService,
	}
}

// CheckHealth performs health check with business logic
func (s *healthService) CheckHealth() (*model.HealthResponse, error) {
	logger.Info("Starting health check...")

	// Call repository to get health status
	health, err := s.healthRepo.GetHealthStatus()
	if err != nil {
		logger.Error("Health check failed: ", err)
		return nil, err
	}

	// Business logic: Add additional checks here
	// For example: check external services, validate configurations, etc.

	// Log successful health check
	logger.Info("Health check completed successfully")

	return health, nil
}

// GetVehicleStatus returns current vehicle status
func (s *vehicleService) GetVehicleStatus(vehicleID string) (*model.VehicleStatus, error) {
	logger.Info("Getting vehicle status for: ", vehicleID)

	// Business logic: validate vehicle ID
	if vehicleID == "" {
		return nil, errors.New("vehicle_id is required")
	}

	// Try cache first (Cache-Aside pattern)
	ctx := context.Background()
	cachedStatus, err := s.cacheService.GetVehicleStatus(ctx, vehicleID)
	if err != nil {
		logger.Error("Cache read error: ", err)
		// Continue to database on cache error
	} else if cachedStatus != nil {
		logger.Info("Cache hit for vehicle: ", vehicleID)
		return cachedStatus, nil
	}

	// Cache miss - get from database
	logger.Info("Cache miss for vehicle: ", vehicleID)
	status, err := s.vehicleRepo.GetVehicleStatus(vehicleID)
	if err != nil {
		logger.Error("Failed to get vehicle status: ", err)
		return nil, err
	}

	// Store in cache for 5 minutes
	cacheTTL := 5 * time.Minute
	if err := s.cacheService.SetVehicleStatus(ctx, vehicleID, status, cacheTTL); err != nil {
		logger.Error("Failed to cache vehicle status: ", err)
		// Don't fail the request if caching fails
	} else {
		logger.Info("Cached vehicle status for: ", vehicleID)
	}

	logger.Info("Successfully retrieved vehicle status")
	return status, nil
}

// GetTripHistory returns trip history for specified hours
func (s *vehicleService) GetTripHistory(vehicleID string, hours int) (*model.TripHistoryResponse, error) {
	logger.Info("Getting trip history for vehicle: ", vehicleID, " hours: ", hours)

	// Business logic: validate parameters
	if vehicleID == "" {
		return nil, errors.New("vehicle_id is required")
	}

	if hours <= 0 {
		hours = 24 // Default to 24 hours
	}

	if hours > 24 { // Max 24 hours
		hours = 24
	}

	trips, err := s.vehicleRepo.GetTripHistory(vehicleID, hours)
	if err != nil {
		logger.Error("Failed to get trip history: ", err)
		return nil, err
	}

	response := &model.TripHistoryResponse{
		VehicleID: vehicleID,
		Trips:     trips,
		Count:     len(trips),
	}

	logger.Info("Successfully retrieved trip history, count: ", len(trips))
	return response, nil
}

// IngestVehicleData processes incoming vehicle data
func (s *vehicleService) IngestVehicleData(data *model.VehicleData) error {
	logger.Info("Ingesting vehicle data for: ", data.VehicleID)

	// Initialize parser and trip manager
	parser := NewVehicleDataParser()
	tripManager := NewTripManager(s.vehicleRepo, s.cacheService)

	// Ensure status object exists
	if data.Status == nil {
		data.Status = make(map[string]interface{})
	}

	// Parse and validate the incoming data
	parsedData, err := parser.ParseAndValidate(map[string]interface{}{
		"vehicle_id": data.VehicleID,
		"status":     data.Status,
	})
	if err != nil {
		return errors.New("data validation failed: " + err.Error())
	}

	// Log parsed location for debugging
	logger.Info("Parsed current location: [", parsedData.Location[0], ",", parsedData.Location[1], "]")

	// Update vehicle status in database first
	err = s.vehicleRepo.IngestVehicleData(data)
	if err != nil {
		return errors.New("failed to ingest vehicle data: " + err.Error())
	}

	// Process trip management
	err = tripManager.ProcessVehicleData(parsedData)
	if err != nil {
		logger.Error("Trip processing failed: ", err)
		// Don't fail the entire request if trip processing fails
	}

	// Invalidate cache after successful update
	err = tripManager.InvalidateCache(data.VehicleID)
	if err != nil {
		logger.Error("Cache invalidation failed: ", err)
		// Don't fail the request if cache invalidation fails
	}

	logger.Info("Successfully ingested vehicle data")
	return nil
}

// calculateDistance calculates distance between two coordinates using Haversine formula
func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371 // Earth's radius in kilometers

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
