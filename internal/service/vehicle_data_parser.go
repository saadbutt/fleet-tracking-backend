package service

import (
	"errors"
	"time"
)

// VehicleDataParser handles parsing and validation of vehicle data
type VehicleDataParser struct{}

// ParsedVehicleData represents validated and parsed vehicle data
type ParsedVehicleData struct {
	VehicleID         string
	Location          [2]float64
	Speed             float64
	Timestamp         time.Time
	HasValidLocation  bool
	HasValidSpeed     bool
	HasValidTimestamp bool
}

// NewVehicleDataParser creates a new vehicle data parser
func NewVehicleDataParser() *VehicleDataParser {
	return &VehicleDataParser{}
}

// ParseAndValidate parses and validates incoming vehicle data
func (p *VehicleDataParser) ParseAndValidate(data map[string]interface{}) (*ParsedVehicleData, error) {
	// Extract vehicle ID
	vehicleID, err := p.extractVehicleID(data)
	if err != nil {
		return nil, err
	}

	// Extract and validate status object
	status, err := p.extractStatus(data)
	if err != nil {
		return nil, err
	}

	// Parse location
	location, hasValidLocation := p.parseLocation(status)

	// Parse speed
	speed, hasValidSpeed := p.parseSpeed(status)

	// Parse timestamp
	timestamp, hasValidTimestamp := p.parseTimestamp(status)

	return &ParsedVehicleData{
		VehicleID:         vehicleID,
		Location:          location,
		Speed:             speed,
		Timestamp:         timestamp,
		HasValidLocation:  hasValidLocation,
		HasValidSpeed:     hasValidSpeed,
		HasValidTimestamp: hasValidTimestamp,
	}, nil
}

// extractVehicleID extracts and validates vehicle ID
func (p *VehicleDataParser) extractVehicleID(data map[string]interface{}) (string, error) {
	vehicleID, exists := data["vehicle_id"]
	if !exists {
		return "", errors.New("vehicle_id is required")
	}

	vehicleIDStr, ok := vehicleID.(string)
	if !ok || vehicleIDStr == "" {
		return "", errors.New("vehicle_id must be a non-empty string")
	}

	return vehicleIDStr, nil
}

// extractStatus extracts and validates status object
func (p *VehicleDataParser) extractStatus(data map[string]interface{}) (map[string]interface{}, error) {
	status, exists := data["status"]
	if !exists {
		return nil, errors.New("status object is required")
	}

	statusMap, ok := status.(map[string]interface{})
	if !ok {
		return nil, errors.New("status must be an object")
	}

	return statusMap, nil
}

// parseLocation parses location coordinates from status
func (p *VehicleDataParser) parseLocation(status map[string]interface{}) ([2]float64, bool) {
	location, exists := status["location"]
	if !exists {
		return [2]float64{0, 0}, false
	}

	// Handle both []interface{} and []float64 cases
	var locArray []interface{}
	switch v := location.(type) {
	case []interface{}:
		locArray = v
	case []float64:
		// Convert []float64 to []interface{}
		locArray = make([]interface{}, len(v))
		for i, val := range v {
			locArray[i] = val
		}
	default:
		return [2]float64{0, 0}, false
	}

	if len(locArray) != 2 {
		return [2]float64{0, 0}, false
	}

	lat, latOk := locArray[0].(float64)
	lng, lngOk := locArray[1].(float64)

	if !latOk || !lngOk {
		return [2]float64{0, 0}, false
	}

	return [2]float64{lat, lng}, true
}

// parseSpeed parses speed from status
func (p *VehicleDataParser) parseSpeed(status map[string]interface{}) (float64, bool) {
	speed, exists := status["speed"]
	if !exists {
		return 0, false
	}

	speedFloat, ok := speed.(float64)
	if !ok {
		return 0, false
	}

	return speedFloat, true
}

// parseTimestamp parses timestamp from status
func (p *VehicleDataParser) parseTimestamp(status map[string]interface{}) (time.Time, bool) {
	timestamp, exists := status["timestamp"]
	if !exists {
		return time.Time{}, false
	}

	timestampStr, ok := timestamp.(string)
	if !ok {
		return time.Time{}, false
	}

	parsedTime, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return time.Time{}, false
	}

	return parsedTime, true
}
