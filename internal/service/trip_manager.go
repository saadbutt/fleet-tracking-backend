package service

import (
	"context"
	"errors"
	"time"

	"fleet-tracking/internal/repository"
	"fleet-tracking/pkg/logger"
)

// TripManager handles trip creation and updates
type TripManager struct {
	vehicleRepo  repository.VehicleRepository
	cacheService CacheService
}

// NewTripManager creates a new trip manager
func NewTripManager(vehicleRepo repository.VehicleRepository, cacheService CacheService) *TripManager {
	return &TripManager{
		vehicleRepo:  vehicleRepo,
		cacheService: cacheService,
	}
}

// ProcessVehicleData processes vehicle data and manages trip lifecycle
func (tm *TripManager) ProcessVehicleData(parsedData *ParsedVehicleData) error {
	logger.Info("Processing vehicle data for: ", parsedData.VehicleID)

	// Check if there's an open trip
	hasOpenTrip, err := tm.vehicleRepo.HasOpenTrip(parsedData.VehicleID)
	if err != nil {
		return errors.New("failed to check for open trip: " + err.Error())
	}

	// Determine trip action based on current state and speed
	tripAction := tm.determineTripAction(hasOpenTrip, parsedData.Speed, parsedData.HasValidSpeed)

	// Execute the determined action
	switch tripAction {
	case TripActionCreateNew:
		return tm.createNewTrip(parsedData)
	case TripActionUpdateExisting:
		return tm.updateExistingTrip(parsedData)
	case TripActionUpdateVehicleOnly:
		return tm.updateVehicleStatusOnly(parsedData)
	default:
		return tm.updateVehicleStatusOnly(parsedData)
	}
}

// TripAction represents the action to take for trip management
type TripAction int

const (
	TripActionCreateNew TripAction = iota
	TripActionUpdateExisting
	TripActionUpdateVehicleOnly
)

// determineTripAction determines what action to take based on current state
func (tm *TripManager) determineTripAction(hasOpenTrip bool, speed float64, hasValidSpeed bool) TripAction {
	// If no valid speed data, just update vehicle status
	if !hasValidSpeed {
		return TripActionUpdateVehicleOnly
	}

	// If there's an open trip
	if hasOpenTrip {
		// If speed is 0, vehicle is stationary - keep trip open
		if speed == 0 {
			logger.Info("Vehicle is stationary (speed=0), keeping trip open")
			return TripActionUpdateVehicleOnly
		}
		// Vehicle is moving, update the trip
		logger.Info("Found open trip, will update it")
		return TripActionUpdateExisting
	}

	// No open trip
	// If speed > 0, create new trip
	if speed > 0 {
		logger.Info("No open trip found, will create new one (speed > 0)")
		return TripActionCreateNew
	}

	// Speed is 0 and no open trip - just update vehicle status
	logger.Info("No open trip and speed=0, just updating vehicle status")
	return TripActionUpdateVehicleOnly
}

// createNewTrip creates a new trip record
func (tm *TripManager) createNewTrip(parsedData *ParsedVehicleData) error {
	if !parsedData.HasValidTimestamp {
		parsedData.Timestamp = time.Now()
	}

	err := tm.vehicleRepo.CreateTripRecord(parsedData.VehicleID, parsedData.Timestamp, parsedData.Speed)
	if err != nil {
		return errors.New("failed to create new trip: " + err.Error())
	}

	logger.Info("Created new trip for vehicle: ", parsedData.VehicleID, " with initial speed: ", parsedData.Speed)
	return nil
}

// updateExistingTrip updates an existing trip with new data
func (tm *TripManager) updateExistingTrip(parsedData *ParsedVehicleData) error {
	// Get previous location and speed for calculation
	previousLocation, previousSpeed, err := tm.vehicleRepo.GetLastTripLocation(parsedData.VehicleID)
	if err != nil {
		return errors.New("failed to get last trip location: " + err.Error())
	}

	logger.Info("Got previous location: [", previousLocation[0], ",", previousLocation[1], "] speed: ", previousSpeed)

	// Calculate mileage and average speed
	mileage := calculateDistance(
		previousLocation[0], previousLocation[1],
		parsedData.Location[0], parsedData.Location[1],
	)
	avgSpeed := (previousSpeed + parsedData.Speed) / 2

	logger.Info("Trip update - Previous: [", previousLocation[0], ",", previousLocation[1], "] Current: [", parsedData.Location[0], ",", parsedData.Location[1], "] Mileage: ", mileage)

	// Update the trip record
	if !parsedData.HasValidTimestamp {
		parsedData.Timestamp = time.Now()
	}

	err = tm.vehicleRepo.UpdateTripRecord(parsedData.VehicleID, parsedData.Timestamp, mileage, avgSpeed)
	if err != nil {
		return errors.New("failed to update trip: " + err.Error())
	}

	logger.Info("Updated trip for vehicle: ", parsedData.VehicleID, " mileage: ", mileage, " avg_speed: ", avgSpeed)
	return nil
}

// updateVehicleStatusOnly updates only the vehicle status without trip changes
func (tm *TripManager) updateVehicleStatusOnly(parsedData *ParsedVehicleData) error {
	logger.Info("Updating vehicle status only for: ", parsedData.VehicleID)

	// For stationary vehicles or vehicles with no open trips,
	// we just log the status update. The vehicle data is already
	// ingested in the main service layer.

	logger.Info("Vehicle status updated for: ", parsedData.VehicleID,
		" location: [", parsedData.Location[0], ",", parsedData.Location[1], "]",
		" speed: ", parsedData.Speed)

	return nil
}

// InvalidateCache invalidates the cache for a vehicle
func (tm *TripManager) InvalidateCache(vehicleID string) error {
	ctx := context.Background()
	if err := tm.cacheService.InvalidateVehicleStatus(ctx, vehicleID); err != nil {
		logger.Error("Failed to invalidate cache: ", err)
		return err
	}

	logger.Info("Invalidated cache for vehicle: ", vehicleID)
	return nil
}
