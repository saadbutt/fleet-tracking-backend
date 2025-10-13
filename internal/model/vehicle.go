package model

import "time"

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// Vehicle represents a vehicle in the database
type Vehicle struct {
	ID          string    `json:"id" db:"id"`
	PlateNumber string    `json:"plate_number" db:"plate_number"`
	LastStatus  []byte    `json:"last_status" db:"last_status"` // JSONB in database
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// VehicleStatus represents current vehicle status
type VehicleStatus struct {
	VehicleID   string                 `json:"vehicle_id"`
	PlateNumber string                 `json:"plate_number"`
	Status      map[string]interface{} `json:"status"`
	LastUpdated time.Time              `json:"last_updated"`
}

// Trip represents a vehicle trip
type Trip struct {
	ID        string     `json:"id" db:"id"`
	VehicleID string     `json:"vehicle_id" db:"vehicle_id"`
	StartTime time.Time  `json:"start_time" db:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty" db:"end_time"`
	Mileage   float64    `json:"mileage" db:"mileage"`
	AvgSpeed  float64    `json:"avg_speed" db:"avg_speed"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// VehicleData represents incoming vehicle data from API (for ingestion)
type VehicleData struct {
	VehicleID   string                 `json:"vehicle_id"`             // Input field (not database ID)
	PlateNumber string                 `json:"plate_number,omitempty"` // Optional
	Status      map[string]interface{} `json:"status"`                 // Input JSON data (includes timestamp)
}

// TripHistoryResponse represents trip history response
type TripHistoryResponse struct {
	VehicleID string `json:"vehicle_id"`
	Trips     []Trip `json:"trips"`
	Count     int    `json:"count"`
}
