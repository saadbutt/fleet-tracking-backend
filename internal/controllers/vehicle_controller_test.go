package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fleet-tracking/internal/auth"
	"fleet-tracking/internal/model"
)

// MockVehicleService implements VehicleService interface for testing
type MockVehicleService struct {
	vehicles map[string]*model.Vehicle
	trips    map[string][]model.Trip
	errors   map[string]error
}

func NewMockVehicleService() *MockVehicleService {
	return &MockVehicleService{
		vehicles: make(map[string]*model.Vehicle),
		trips:    make(map[string][]model.Trip),
		errors:   make(map[string]error),
	}
}

func (m *MockVehicleService) SetError(method string, err error) {
	m.errors[method] = err
}

func (m *MockVehicleService) GetVehicleStatus(vehicleID string) (*model.VehicleStatus, error) {
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

func (m *MockVehicleService) GetTripHistory(vehicleID string, hours int) (*model.TripHistoryResponse, error) {
	if err, exists := m.errors["GetTripHistory"]; exists {
		return nil, err
	}
	var trips []model.Trip
	if t, exists := m.trips[vehicleID]; exists {
		trips = t
	}
	return &model.TripHistoryResponse{
		Trips: trips,
		Count: len(trips),
	}, nil
}

func (m *MockVehicleService) IngestVehicleData(data *model.VehicleData) error {
	if err, exists := m.errors["IngestVehicleData"]; exists {
		return err
	}
	// Simulate successful ingestion
	return nil
}

func TestVehicleController_GetVehicleStatus(t *testing.T) {
	tests := []struct {
		name           string
		vehicleID      string
		setupService   func(*MockVehicleService)
		setupContext   func() context.Context
		wantStatusCode int
		wantError      bool
		description    string
	}{
		{
			name:      "successful vehicle status retrieval",
			vehicleID: "test-vehicle-1",
			setupService: func(service *MockVehicleService) {
				service.vehicles["test-vehicle-1"] = &model.Vehicle{
					ID:          "test-vehicle-1",
					PlateNumber: "ABC-123",
					LastStatus:  []byte(`{"location":[25.2048,55.2708],"speed":60.5,"timestamp":"2025-10-13T03:00:00Z"}`),
					UpdatedAt:   time.Now(),
				}
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusOK,
			wantError:      false,
			description:    "Should return vehicle status for valid vehicle ID",
		},
		{
			name:      "vehicle not found",
			vehicleID: "non-existent-vehicle",
			setupService: func(service *MockVehicleService) {
				service.SetError("GetVehicleStatus", errors.New("vehicle not found"))
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      true,
			description:    "Should return 404 for non-existent vehicle",
		},
		{
			name:      "missing vehicle ID parameter",
			vehicleID: "",
			setupService: func(service *MockVehicleService) {
				// No setup needed
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
			description:    "Should return 400 for missing vehicle ID",
		},
		{
			name:      "service error",
			vehicleID: "test-vehicle-2",
			setupService: func(service *MockVehicleService) {
				service.SetError("GetVehicleStatus", errors.New("database connection failed"))
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      true,
			description:    "Should return 500 for service errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := NewMockVehicleService()
			tt.setupService(mockService)

			controller := NewVehicleController(mockService)

			// Create request
			req := httptest.NewRequest("GET", "/api/vehicle/status", nil)
			if tt.vehicleID != "" {
				q := req.URL.Query()
				q.Add("vehicle_id", tt.vehicleID)
				req.URL.RawQuery = q.Encode()
			}
			req = req.WithContext(tt.setupContext())

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute
			controller.GetVehicleStatus(w, req)

			// Verify status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("GetVehicleStatus() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Verify response body for error cases
			if tt.wantError {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
					return
				}

				if _, hasError := response["error"]; !hasError {
					t.Error("Expected error field in response")
				}
			} else {
				// Verify successful response structure
				var response model.VehicleStatus
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal success response: %v", err)
					return
				}

				if response.VehicleID != tt.vehicleID {
					t.Errorf("Expected vehicle ID %v, got %v", tt.vehicleID, response.VehicleID)
				}
			}
		})
	}
}

func TestVehicleController_GetTripHistory(t *testing.T) {
	tests := []struct {
		name           string
		vehicleID      string
		hours          string
		setupService   func(*MockVehicleService)
		setupContext   func() context.Context
		wantStatusCode int
		wantError      bool
		expectedTrips  int
		description    string
	}{
		{
			name:      "successful trip history retrieval",
			vehicleID: "test-vehicle-1",
			hours:     "24",
			setupService: func(service *MockVehicleService) {
				service.trips["test-vehicle-1"] = []model.Trip{
					{
						ID:        "trip-1",
						VehicleID: "test-vehicle-1",
						StartTime: time.Now().Add(-2 * time.Hour),
						EndTime:   func() *time.Time { t := time.Now().Add(-1 * time.Hour); return &t }(),
						Mileage:   10.5,
						AvgSpeed:  65.0,
						CreatedAt: time.Now().Add(-2 * time.Hour),
					},
					{
						ID:        "trip-2",
						VehicleID: "test-vehicle-1",
						StartTime: time.Now().Add(-1 * time.Hour),
						EndTime:   nil, // Open trip
						Mileage:   5.2,
						AvgSpeed:  45.0,
						CreatedAt: time.Now().Add(-1 * time.Hour),
					},
				}
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusOK,
			wantError:      false,
			expectedTrips:  2,
			description:    "Should return trip history for valid vehicle ID",
		},
		{
			name:      "no trips found",
			vehicleID: "test-vehicle-2",
			hours:     "24",
			setupService: func(service *MockVehicleService) {
				// No trips setup - empty result
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusOK,
			wantError:      false,
			expectedTrips:  0,
			description:    "Should return empty array when no trips found",
		},
		{
			name:      "missing vehicle ID",
			vehicleID: "",
			hours:     "24",
			setupService: func(service *MockVehicleService) {
				// No setup needed
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
			expectedTrips:  0,
			description:    "Should return 400 for missing vehicle ID",
		},
		{
			name:      "invalid hours parameter (ignored)",
			vehicleID: "test-vehicle-3",
			hours:     "invalid",
			setupService: func(service *MockVehicleService) {
				// No setup needed - controller ignores invalid hours and uses default 24
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusOK,
			wantError:      false,
			expectedTrips:  0,
			description:    "Should ignore invalid hours parameter and use default 24",
		},
		{
			name:      "service error",
			vehicleID: "test-vehicle-4",
			hours:     "24",
			setupService: func(service *MockVehicleService) {
				service.SetError("GetTripHistory", errors.New("database connection failed"))
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      true,
			expectedTrips:  0,
			description:    "Should return 500 for service errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := NewMockVehicleService()
			tt.setupService(mockService)

			controller := NewVehicleController(mockService)

			// Create request
			req := httptest.NewRequest("GET", "/api/vehicle/trips", nil)
			q := req.URL.Query()
			if tt.vehicleID != "" {
				q.Add("vehicle_id", tt.vehicleID)
			}
			if tt.hours != "" {
				q.Add("hours", tt.hours)
			}
			req.URL.RawQuery = q.Encode()
			req = req.WithContext(tt.setupContext())

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute
			controller.GetTripHistory(w, req)

			// Verify status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("GetTripHistory() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Verify response body
			if tt.wantError {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
					return
				}

				if _, hasError := response["error"]; !hasError {
					t.Error("Expected error field in response")
				}
			} else {
				var response struct {
					Trips []model.Trip `json:"trips"`
					Count int          `json:"count"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal success response: %v", err)
					return
				}

				if len(response.Trips) != tt.expectedTrips {
					t.Errorf("Expected %d trips, got %d", tt.expectedTrips, len(response.Trips))
				}

				if response.Count != tt.expectedTrips {
					t.Errorf("Expected count %d, got %d", tt.expectedTrips, response.Count)
				}
			}
		})
	}
}

func TestVehicleController_IngestVehicleData(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupService   func(*MockVehicleService)
		setupContext   func() context.Context
		wantStatusCode int
		wantError      bool
		description    string
	}{
		{
			name: "successful vehicle data ingestion",
			requestBody: map[string]interface{}{
				"vehicle_id": "test-vehicle-1",
				"status": map[string]interface{}{
					"location":  []interface{}{25.2048, 55.2708},
					"speed":     60.5,
					"timestamp": "2025-10-13T03:00:00Z",
				},
			},
			setupService: func(service *MockVehicleService) {
				// No error setup - should succeed
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusCreated,
			wantError:      false,
			description:    "Should successfully ingest valid vehicle data",
		},
		{
			name:        "invalid JSON",
			requestBody: "invalid json",
			setupService: func(service *MockVehicleService) {
				// No setup needed
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
			description:    "Should return 400 for invalid JSON",
		},
		{
			name: "missing vehicle_id",
			requestBody: map[string]interface{}{
				"status": map[string]interface{}{
					"location":  []interface{}{25.2048, 55.2708},
					"speed":     60.5,
					"timestamp": "2025-10-13T03:00:00Z",
				},
			},
			setupService: func(service *MockVehicleService) {
				// No setup needed
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
			description:    "Should return 400 for missing vehicle_id",
		},
		{
			name: "missing status",
			requestBody: map[string]interface{}{
				"vehicle_id": "test-vehicle-2",
			},
			setupService: func(service *MockVehicleService) {
				// No setup needed
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
			description:    "Should return 400 for missing status",
		},
		{
			name: "service error during ingestion",
			requestBody: map[string]interface{}{
				"vehicle_id": "test-vehicle-3",
				"status": map[string]interface{}{
					"location":  []interface{}{25.2048, 55.2708},
					"speed":     60.5,
					"timestamp": "2025-10-13T03:00:00Z",
				},
			},
			setupService: func(service *MockVehicleService) {
				service.SetError("IngestVehicleData", errors.New("database connection failed"))
			},
			setupContext: func() context.Context {
				return auth.WithAuthentication(context.Background(), true)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
			description:    "Should return 500 for service errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := NewMockVehicleService()
			tt.setupService(mockService)

			controller := NewVehicleController(mockService)

			// Create request body
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			// Create request
			req := httptest.NewRequest("POST", "/api/vehicle/ingest", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tt.setupContext())

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute
			controller.IngestVehicleData(w, req)

			// Verify status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("IngestVehicleData() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Verify response body
			if tt.wantError {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
					return
				}

				if _, hasError := response["error"]; !hasError {
					t.Error("Expected error field in response")
				}
			} else {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal success response: %v", err)
					return
				}

				if _, hasMessage := response["message"]; !hasMessage {
					t.Error("Expected message field in success response")
				}
			}
		})
	}
}
