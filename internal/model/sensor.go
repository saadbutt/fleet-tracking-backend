package model

import (
	"math/rand"
	"time"
)

// SensorData represents incoming sensor data from vehicle
type SensorData struct {
	VehicleID string     `json:"vehicle_id"`
	Timestamp time.Time  `json:"timestamp"`
	Location  [2]float64 `json:"location"` // [latitude, longitude]
	Speed     float64    `json:"speed"`    // km/h
}

// SensorMessage represents a message sent through the channel
type SensorMessage struct {
	Data      *SensorData
	Error     error
	Processed bool
}

// GenerateRandomSensorData creates realistic sensor data for simulation
func GenerateRandomSensorData(vehicleID string) *SensorData {
	baseLat := 25.2048
	baseLng := 55.2708

	// Seed random number generator with current time for better randomness
	rand.Seed(time.Now().UnixNano())

	// Add randomness to location (larger variation for realistic movement)
	lat := baseLat + (rand.Float64()-0.5)*0.1 // ±0.05 degrees (~5.5 km)
	lng := baseLng + (rand.Float64()-0.5)*0.1 // ±0.05 degrees (~5.5 km)

	// Realistic speed range (0-120 km/h)
	speed := rand.Float64() * 120

	return &SensorData{
		VehicleID: vehicleID,
		Timestamp: time.Now().UTC(),
		Location:  [2]float64{lat, lng},
		Speed:     speed,
	}
}
