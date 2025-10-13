package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"fleet-tracking/internal/model"
)

// MockVehicleRepository implements VehicleRepository interface for testing
type MockVehicleRepository struct {
	vehicles map[string]*model.Vehicle
	trips    []model.Trip
	errors   map[string]error // Map of method names to errors
}

func NewMockVehicleRepository() *MockVehicleRepository {
	return &MockVehicleRepository{
		vehicles: make(map[string]*model.Vehicle),
		trips:    make([]model.Trip, 0),
		errors:   make(map[string]error),
	}
}

func (m *MockVehicleRepository) SetError(method string, err error) {
	m.errors[method] = err
}

func (m *MockVehicleRepository) GetVehicleStatus(vehicleID string) (*model.VehicleStatus, error) {
	if err, exists := m.errors["GetVehicleStatus"]; exists {
		return nil, err
	}
	if vehicle, exists := m.vehicles[vehicleID]; exists {
		// Convert Vehicle to VehicleStatus
		var status map[string]interface{}
		json.Unmarshal(vehicle.LastStatus, &status)
		return &model.VehicleStatus{
			VehicleID:   vehicle.ID,
			PlateNumber: vehicle.PlateNumber,
			Status:      status,
			LastUpdated: vehicle.UpdatedAt,
		}, nil
	}
	return nil, errors.New("vehicle not found")
}

func (m *MockVehicleRepository) GetVehicleByID(id string) (*model.Vehicle, error) {
	if err, exists := m.errors["GetVehicleByID"]; exists {
		return nil, err
	}
	if vehicle, exists := m.vehicles[id]; exists {
		return vehicle, nil
	}
	return nil, errors.New("vehicle not found")
}

func (m *MockVehicleRepository) IngestVehicleData(data *model.VehicleData) error {
	if err, exists := m.errors["IngestVehicleData"]; exists {
		return err
	}
	// Simulate updating vehicle data
	vehicle := &model.Vehicle{
		ID:          data.VehicleID,
		PlateNumber: "ABC-123", // Mock plate number
		LastStatus:  []byte(`{"location":[25.2048,55.2708],"speed":60.5,"timestamp":"2025-10-13T03:00:00Z"}`),
		UpdatedAt:   time.Now(),
	}
	m.vehicles[data.VehicleID] = vehicle
	return nil
}

func (m *MockVehicleRepository) GetTripHistory(vehicleID string, hours int) ([]model.Trip, error) {
	if err, exists := m.errors["GetTripHistory"]; exists {
		return nil, err
	}
	return m.trips, nil
}

func (m *MockVehicleRepository) CreateTripRecord(vehicleID string, startTime time.Time, initialSpeed float64) error {
	if err, exists := m.errors["CreateTripRecord"]; exists {
		return err
	}
	trip := model.Trip{
		ID:        "trip-" + vehicleID,
		VehicleID: vehicleID,
		StartTime: startTime,
		EndTime:   nil,
		Mileage:   0,
		AvgSpeed:  initialSpeed,
		CreatedAt: time.Now(),
	}
	m.trips = append(m.trips, trip)
	return nil
}

func (m *MockVehicleRepository) UpdateTripRecord(vehicleID string, endTime time.Time, mileage, avgSpeed float64) error {
	if err, exists := m.errors["UpdateTripRecord"]; exists {
		return err
	}
	// Find and update the last open trip
	for i := len(m.trips) - 1; i >= 0; i-- {
		if m.trips[i].VehicleID == vehicleID && m.trips[i].EndTime == nil {
			m.trips[i].EndTime = &endTime
			m.trips[i].Mileage = mileage
			m.trips[i].AvgSpeed = avgSpeed
			break
		}
	}
	return nil
}

func (m *MockVehicleRepository) HasOpenTrip(vehicleID string) (bool, error) {
	if err, exists := m.errors["HasOpenTrip"]; exists {
		return false, err
	}
	// Check if there's an open trip
	for i := len(m.trips) - 1; i >= 0; i-- {
		if m.trips[i].VehicleID == vehicleID && m.trips[i].EndTime == nil {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockVehicleRepository) GetLastTripLocation(vehicleID string) ([2]float64, float64, error) {
	if err, exists := m.errors["GetLastTripLocation"]; exists {
		return [2]float64{0, 0}, 0, err
	}
	// Return mock location and speed
	return [2]float64{25.2048, 55.2708}, 60.5, nil
}

func (m *MockVehicleRepository) GetFirstVehicleID() (string, error) {
	if err, exists := m.errors["GetFirstVehicleID"]; exists {
		return "", err
	}
	// Return first vehicle ID from mock data
	for id := range m.vehicles {
		return id, nil
	}
	return "", errors.New("no vehicles found")
}

// MockCacheService implements CacheService interface for testing
type MockCacheService struct {
	cache  map[string]interface{}
	errors map[string]error
}

func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		cache:  make(map[string]interface{}),
		errors: make(map[string]error),
	}
}

func (m *MockCacheService) SetError(method string, err error) {
	m.errors[method] = err
}

func (m *MockCacheService) GetVehicleStatus(ctx context.Context, vehicleID string) (*model.VehicleStatus, error) {
	if err, exists := m.errors["GetVehicleStatus"]; exists {
		return nil, err
	}
	if status, exists := m.cache["vehicle:"+vehicleID]; exists {
		return status.(*model.VehicleStatus), nil
	}
	return nil, errors.New("not found in cache")
}

func (m *MockCacheService) SetVehicleStatus(ctx context.Context, vehicleID string, status *model.VehicleStatus, ttl time.Duration) error {
	if err, exists := m.errors["SetVehicleStatus"]; exists {
		return err
	}
	m.cache["vehicle:"+vehicleID] = status
	return nil
}

func (m *MockCacheService) InvalidateVehicleStatus(ctx context.Context, vehicleID string) error {
	if err, exists := m.errors["InvalidateVehicleStatus"]; exists {
		return err
	}
	delete(m.cache, "vehicle:"+vehicleID)
	return nil
}

func TestVehicleService_IngestVehicleData(t *testing.T) {
	tests := []struct {
		name          string
		input         *model.VehicleData
		setupMocks    func(*MockVehicleRepository, *MockCacheService)
		wantError     bool
		expectedTrips int
		description   string
	}{
		{
			name: "successful new trip creation",
			input: &model.VehicleData{
				VehicleID: "test-vehicle-1",
				Status: map[string]interface{}{
					"location":  []interface{}{25.2048, 55.2708},
					"speed":     60.5,
					"timestamp": "2025-10-13T03:00:00Z",
				},
			},
			setupMocks: func(repo *MockVehicleRepository, cache *MockCacheService) {
				// No open trip exists, should create new one
			},
			wantError:     false,
			expectedTrips: 1,
			description:   "Should create new trip when no open trip exists and speed > 0",
		},
		{
			name: "successful trip update",
			input: &model.VehicleData{
				VehicleID: "test-vehicle-2",
				Status: map[string]interface{}{
					"location":  []interface{}{25.2049, 55.2709},
					"speed":     70.0,
					"timestamp": "2025-10-13T03:01:00Z",
				},
			},
			setupMocks: func(repo *MockVehicleRepository, cache *MockCacheService) {
				// Simulate existing open trip
				repo.trips = append(repo.trips, model.Trip{
					ID:        "existing-trip",
					VehicleID: "test-vehicle-2",
					StartTime: time.Now().Add(-1 * time.Hour),
					EndTime:   nil, // Open trip
					Mileage:   0,
					AvgSpeed:  60.0,
					CreatedAt: time.Now().Add(-1 * time.Hour),
				})
			},
			wantError:     false,
			expectedTrips: 1, // Still 1 trip, but updated
			description:   "Should update existing open trip when speed > 0",
		},
		{
			name: "speed zero with no open trip",
			input: &model.VehicleData{
				VehicleID: "test-vehicle-3",
				Status: map[string]interface{}{
					"location":  []interface{}{25.2048, 55.2708},
					"speed":     0.0,
					"timestamp": "2025-10-13T03:00:00Z",
				},
			},
			setupMocks: func(repo *MockVehicleRepository, cache *MockCacheService) {
				// No open trip exists
			},
			wantError:     false,
			expectedTrips: 0,
			description:   "Should not create trip when speed = 0 and no open trip exists",
		},
		{
			name: "speed zero with open trip",
			input: &model.VehicleData{
				VehicleID: "test-vehicle-4",
				Status: map[string]interface{}{
					"location":  []interface{}{25.2048, 55.2708},
					"speed":     0.0,
					"timestamp": "2025-10-13T03:00:00Z",
				},
			},
			setupMocks: func(repo *MockVehicleRepository, cache *MockCacheService) {
				// Simulate existing open trip
				repo.trips = append(repo.trips, model.Trip{
					ID:        "existing-trip",
					VehicleID: "test-vehicle-4",
					StartTime: time.Now().Add(-1 * time.Hour),
					EndTime:   nil, // Open trip
					Mileage:   0,
					AvgSpeed:  60.0,
					CreatedAt: time.Now().Add(-1 * time.Hour),
				})
			},
			wantError:     false,
			expectedTrips: 1, // Trip remains open, no update
			description:   "Should keep trip open when speed = 0 and open trip exists",
		},
		{
			name: "invalid vehicle ID",
			input: &model.VehicleData{
				VehicleID: "", // Empty vehicle ID
				Status: map[string]interface{}{
					"location":  []interface{}{25.2048, 55.2708},
					"speed":     60.5,
					"timestamp": "2025-10-13T03:00:00Z",
				},
			},
			setupMocks: func(repo *MockVehicleRepository, cache *MockCacheService) {
				// No setup needed for validation error
			},
			wantError:     true,
			expectedTrips: 0,
			description:   "Should return error for empty vehicle ID",
		},
		{
			name: "repository error during ingestion",
			input: &model.VehicleData{
				VehicleID: "test-vehicle-5",
				Status: map[string]interface{}{
					"location":  []interface{}{25.2048, 55.2708},
					"speed":     60.5,
					"timestamp": "2025-10-13T03:00:00Z",
				},
			},
			setupMocks: func(repo *MockVehicleRepository, cache *MockCacheService) {
				repo.SetError("IngestVehicleData", errors.New("database connection failed"))
			},
			wantError:     true,
			expectedTrips: 0,
			description:   "Should return error when repository fails",
		},
		{
			name: "missing status data",
			input: &model.VehicleData{
				VehicleID: "test-vehicle-6",
				Status:    nil, // Missing status
			},
			setupMocks: func(repo *MockVehicleRepository, cache *MockCacheService) {
				// No setup needed
			},
			wantError:     false,
			expectedTrips: 0,
			description:   "Should handle missing status by creating empty map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := NewMockVehicleRepository()
			mockCache := NewMockCacheService()
			tt.setupMocks(mockRepo, mockCache)

			// Create service
			service := NewVehicleService(mockRepo, mockCache)

			// Execute
			err := service.IngestVehicleData(tt.input)

			// Verify error expectation
			if (err != nil) != tt.wantError {
				t.Errorf("IngestVehicleData() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// Verify trip count
			if len(mockRepo.trips) != tt.expectedTrips {
				t.Errorf("Expected %d trips, got %d", tt.expectedTrips, len(mockRepo.trips))
			}

			// Additional validations for successful cases
			if !tt.wantError && tt.expectedTrips > 0 {
				// Verify vehicle data was ingested
				if _, exists := mockRepo.vehicles[tt.input.VehicleID]; !exists {
					t.Error("Expected vehicle to be created/updated in repository")
				}
			}
		})
	}
}

func TestCalculateDistance(t *testing.T) {
	tests := []struct {
		name        string
		lat1, lng1  float64
		lat2, lng2  float64
		want        float64
		tolerance   float64
		description string
	}{
		{
			name:        "same location",
			lat1:        25.2048,
			lng1:        55.2708,
			lat2:        25.2048,
			lng2:        55.2708,
			want:        0.0,
			tolerance:   0.001,
			description: "Distance between same coordinates should be 0",
		},
		{
			name:        "short distance",
			lat1:        25.2048,
			lng1:        55.2708,
			lat2:        25.2049,
			lng2:        55.2709,
			want:        0.014, // Approximately 14 meters
			tolerance:   0.01,
			description: "Short distance calculation",
		},
		{
			name:        "medium distance",
			lat1:        25.2048,
			lng1:        55.2708,
			lat2:        25.2148,
			lng2:        55.2808,
			want:        1.414, // Approximately 1.4 km
			tolerance:   0.1,
			description: "Medium distance calculation",
		},
		{
			name:        "Dubai to Abu Dhabi",
			lat1:        25.2048,
			lng1:        55.2708,
			lat2:        24.4539,
			lng2:        54.3773,
			want:        123.0, // Approximately 123 km
			tolerance:   5.0,
			description: "Long distance calculation (Dubai to Abu Dhabi)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDistance(tt.lat1, tt.lng1, tt.lat2, tt.lng2)

			if got < tt.want-tt.tolerance || got > tt.want+tt.tolerance {
				t.Errorf("calculateDistance() = %v, want %v ± %v (%s)",
					got, tt.want, tt.tolerance, tt.description)
			}
		})
	}
}
