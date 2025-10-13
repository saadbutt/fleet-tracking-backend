package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"fleet-tracking/internal/model"
	"fleet-tracking/pkg/logger"
)

// SimulatorService handles the internal simulated sensor data stream
type SimulatorService interface {
	StartSimulation(ctx context.Context, vehicleID string)
	StopSimulation(vehicleID string)
	IsRunning(vehicleID string) bool
}

// simulatorService implements SimulatorService
type simulatorService struct {
	vehicleService VehicleService
	streams        map[string]context.CancelFunc // vehicleID -> cancel function
	mu             sync.RWMutex                  // protects streams map
	sensorChannel  chan *model.SensorMessage     // channel for sensor data
}

// NewSimulatorService creates a new simulator service
func NewSimulatorService(vehicleService VehicleService) SimulatorService {
	service := &simulatorService{
		vehicleService: vehicleService,
		streams:        make(map[string]context.CancelFunc),
		sensorChannel:  make(chan *model.SensorMessage, 100), // buffered channel
	}

	// Start the message processor
	go service.processSensorMessages(context.Background())

	return service
}

// StartSimulation starts a simulated sensor data stream for a vehicle
func (s *simulatorService) StartSimulation(ctx context.Context, vehicleID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if stream already exists
	if _, exists := s.streams[vehicleID]; exists {
		logger.Info("Simulation already running for vehicle: ", vehicleID)
		return
	}

	// Create context with cancellation
	streamCtx, cancel := context.WithCancel(ctx)
	s.streams[vehicleID] = cancel

	logger.Info("Starting sensor simulation for vehicle: ", vehicleID)

	// Start goroutine for sensor data generation
	go s.generateSensorData(streamCtx, vehicleID)
}

// StopSimulation stops the sensor simulation for a vehicle
func (s *simulatorService) StopSimulation(vehicleID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cancel, exists := s.streams[vehicleID]
	if !exists {
		logger.Info("No active simulation for vehicle: ", vehicleID)
		return
	}

	// Cancel the stream
	cancel()
	delete(s.streams, vehicleID)

	logger.Info("Stopped sensor simulation for vehicle: ", vehicleID)
}

// IsRunning checks if simulation is running for a vehicle
func (s *simulatorService) IsRunning(vehicleID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.streams[vehicleID]
	return exists
}

// generateSensorData generates simulated sensor data every 2 seconds
func (s *simulatorService) generateSensorData(ctx context.Context, vehicleID string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	logger.Info("Sensor data generator started for vehicle: ", vehicleID)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Sensor data generator stopped for vehicle: ", vehicleID)
			return
		case <-ticker.C:
			// Generate random sensor data
			sensorData := model.GenerateRandomSensorData(vehicleID)

			// Send data through channel
			message := &model.SensorMessage{
				Data:      sensorData,
				Error:     nil,
				Processed: false,
			}

			select {
			case s.sensorChannel <- message:
				logger.Info("Generated sensor data for vehicle: ", vehicleID, " at ", sensorData.Timestamp.Format(time.RFC3339))
			case <-ctx.Done():
				return
			default:
				// Channel is full, log warning but continue
				logger.Warn("Sensor channel full, dropping data for vehicle: ", vehicleID)
			}
		}
	}
}

// processSensorMessages processes sensor data from the channel
func (s *simulatorService) processSensorMessages(ctx context.Context) {
	logger.Info("Sensor message processor started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Sensor message processor stopped")
			return
		case message := <-s.sensorChannel:
			if message.Error != nil {
				logger.Error("Sensor message error: ", message.Error)
				continue
			}

			// Process the sensor data
			if err := s.processSensorData(message.Data); err != nil {
				logger.Error("Failed to process sensor data: ", err)
				message.Error = err
			} else {
				message.Processed = true
				logger.Info("Successfully processed sensor data for vehicle: ", message.Data.VehicleID)
			}
		}
	}
}

// processSensorData processes incoming sensor data
func (s *simulatorService) processSensorData(data *model.SensorData) error {
	// Convert sensor data to vehicle data format
	status := map[string]interface{}{
		"location":  data.Location,
		"speed":     data.Speed,
		"timestamp": data.Timestamp.Format(time.RFC3339),
	}

	// Create vehicle data for ingestion
	vehicleData := &model.VehicleData{
		VehicleID: data.VehicleID,
		Status:    status,
	}

	// Log the sensor data (for debugging)
	sensorJSON, _ := json.MarshalIndent(data, "", "  ")
	logger.Info("Processing sensor data: ", string(sensorJSON))

	// Ingest the data (this will update database and invalidate cache)
	if err := s.vehicleService.IngestVehicleData(vehicleData); err != nil {
		return fmt.Errorf("failed to ingest sensor data: %w", err)
	}

	return nil
}
