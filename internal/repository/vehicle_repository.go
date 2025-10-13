package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fleet-tracking/internal/model"
	"fmt"
	"time"
)

// HealthRepository interface for health check operations
type HealthRepository interface {
	GetHealthStatus() (*model.HealthResponse, error)
}

// VehicleRepository interface for vehicle operations
type VehicleRepository interface {
	GetVehicleStatus(vehicleID string) (*model.VehicleStatus, error)
	GetTripHistory(vehicleID string, hours int) ([]model.Trip, error)
	IngestVehicleData(data *model.VehicleData) error
	GetVehicleByID(vehicleID string) (*model.Vehicle, error)
	CreateTripRecord(vehicleID string, startTime time.Time, initialSpeed float64) error
	UpdateTripRecord(vehicleID string, endTime time.Time, mileage, avgSpeed float64) error
	HasOpenTrip(vehicleID string) (bool, error)
	GetLastTripLocation(vehicleID string) ([2]float64, float64, error)
	GetFirstVehicleID() (string, error)
}

// healthRepository implements HealthRepository
type healthRepository struct{}

// vehicleRepository implements VehicleRepository
type vehicleRepository struct {
	db *sql.DB
}

// NewHealthRepository creates a new health repository
func NewHealthRepository() HealthRepository {
	return &healthRepository{}
}

// NewVehicleRepository creates a new vehicle repository
func NewVehicleRepository(db *sql.DB) VehicleRepository {
	return &vehicleRepository{db: db}
}

// GetHealthStatus returns the health status of the application
func (r *healthRepository) GetHealthStatus() (*model.HealthResponse, error) {
	// In a real application, this would check database connectivity,
	// external service health, etc.

	// For now, we'll simulate a successful health check
	health := &model.HealthResponse{
		Status:    "healthy",
		Message:   "All systems operational",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	return health, nil
}

// GetVehicleStatus returns current vehicle status
func (r *vehicleRepository) GetVehicleStatus(vehicleID string) (*model.VehicleStatus, error) {
	query := `
		SELECT id, plate_number, last_status, updated_at 
		FROM vehicle 
		WHERE id = $1`

	var vehicle model.Vehicle

	err := r.db.QueryRow(query, vehicleID).Scan(
		&vehicle.ID,
		&vehicle.PlateNumber,
		&vehicle.LastStatus,
		&vehicle.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("vehicle not found")
		}
		return nil, err
	}

	var status map[string]interface{}
	if len(vehicle.LastStatus) > 0 {
		if err := json.Unmarshal(vehicle.LastStatus, &status); err != nil {
			status = make(map[string]interface{})
		}
	} else {
		status = make(map[string]interface{})
	}

	return &model.VehicleStatus{
		VehicleID:   vehicle.ID,
		PlateNumber: vehicle.PlateNumber,
		Status:      status,
		LastUpdated: vehicle.UpdatedAt,
	}, nil
}

// GetTripHistory returns trip history for specified hours
func (r *vehicleRepository) GetTripHistory(vehicleID string, hours int) ([]model.Trip, error) {
	query := `
		SELECT id, vehicle_id, start_time, end_time, mileage, avg_speed, created_at
		FROM trips 
		WHERE vehicle_id = $1 
		AND start_time >= NOW() - INTERVAL '%d hours'
		ORDER BY start_time DESC`

	rows, err := r.db.Query(fmt.Sprintf(query, hours), vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []model.Trip
	for rows.Next() {
		var trip model.Trip
		err := rows.Scan(
			&trip.ID,
			&trip.VehicleID,
			&trip.StartTime,
			&trip.EndTime,
			&trip.Mileage,
			&trip.AvgSpeed,
			&trip.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		trips = append(trips, trip)
	}

	return trips, nil
}

// IngestVehicleData processes incoming vehicle data
func (r *vehicleRepository) IngestVehicleData(data *model.VehicleData) error {
	// Update existing vehicle status only (vehicle must exist from seeding)
	statusJSON, _ := json.Marshal(data.Status)
	rowsAffected, err := r.db.Exec(`
		UPDATE vehicle 
		SET last_status = $1, updated_at = NOW()
		WHERE id = $2`,
		statusJSON, data.VehicleID)

	if err != nil {
		return err
	}

	// Check if any rows were affected (vehicle exists)
	affected, _ := rowsAffected.RowsAffected()
	if affected == 0 {
		return errors.New("vehicle not found - vehicle must be seeded first")
	}

	return nil
}

// GetVehicleByID returns a vehicle by ID
func (r *vehicleRepository) GetVehicleByID(vehicleID string) (*model.Vehicle, error) {
	query := `SELECT id, plate_number, last_status, created_at, updated_at FROM vehicle WHERE id = $1`

	var vehicle model.Vehicle
	err := r.db.QueryRow(query, vehicleID).Scan(
		&vehicle.ID,
		&vehicle.PlateNumber,
		&vehicle.LastStatus,
		&vehicle.CreatedAt,
		&vehicle.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("vehicle not found")
		}
		return nil, err
	}

	return &vehicle, nil
}

// CreateTripRecord creates a new trip record with start_time and initial speed
func (r *vehicleRepository) CreateTripRecord(vehicleID string, startTime time.Time, initialSpeed float64) error {
	_, err := r.db.Exec(`
		INSERT INTO trips (id, vehicle_id, start_time, end_time, mileage, avg_speed, created_at)
		VALUES (uuid_generate_v4(), $1, $2, NULL, 0, $3, NOW())`,
		vehicleID, startTime, initialSpeed)

	if err != nil {
		return err
	}

	return nil
}

// UpdateTripRecord updates the most recent open trip for a vehicle
func (r *vehicleRepository) UpdateTripRecord(vehicleID string, endTime time.Time, mileage, avgSpeed float64) error {
	// First, find the most recent open trip ID
	var tripID string
	err := r.db.QueryRow(`
		SELECT id FROM trips 
		WHERE vehicle_id = $1 AND end_time IS NULL 
		ORDER BY start_time DESC 
		LIMIT 1`,
		vehicleID).Scan(&tripID)

	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("no open trip found to update")
		}
		return err
	}

	// Update the specific trip
	result, err := r.db.Exec(`
		UPDATE trips 
		SET end_time = $1, mileage = $2, avg_speed = $3, updated_at = NOW()
		WHERE id = $4`,
		endTime, mileage, avgSpeed, tripID)

	if err != nil {
		return err
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("failed to update trip")
	}

	return nil
}

// HasOpenTrip checks if there's an open trip for a vehicle
func (r *vehicleRepository) HasOpenTrip(vehicleID string) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM trips 
		WHERE vehicle_id = $1 AND end_time IS NULL`,
		vehicleID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetLastTripLocation gets the last location and speed from the most recent open trip
func (r *vehicleRepository) GetLastTripLocation(vehicleID string) ([2]float64, float64, error) {
	var location [2]float64
	var speed float64

	// Get the last location from the vehicle's last_status
	err := r.db.QueryRow(`
		SELECT 
			(last_status->'location'->>0)::float,
			(last_status->'location'->>1)::float,
			(last_status->>'speed')::float
		FROM vehicle 
		WHERE id = $1`,
		vehicleID).Scan(&location[0], &location[1], &speed)

	if err != nil {
		return [2]float64{0, 0}, 0, err
	}

	return location, speed, nil
}

// GetFirstVehicleID returns the ID of the first vehicle in the database
func (r *vehicleRepository) GetFirstVehicleID() (string, error) {
	var vehicleID string
	err := r.db.QueryRow(`
		SELECT id 
		FROM vehicle 
		ORDER BY created_at ASC 
		LIMIT 1`).Scan(&vehicleID)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("no vehicles found in database")
		}
		return "", err
	}

	return vehicleID, nil
}
