-- Complete Database Schema Migration
-- This file contains the complete database setup for the Fleet Tracking API
-- Run this once to set up the entire database schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create vehicle table
CREATE TABLE vehicle (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plate_number TEXT NOT NULL UNIQUE,
    last_status JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create trips table
CREATE TABLE trips (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id UUID NOT NULL REFERENCES vehicle(id) ON DELETE CASCADE,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    mileage FLOAT,
    avg_speed FLOAT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for query optimization
-- 1. Composite index for GetTripHistory query:
--    SELECT * FROM trips WHERE vehicle_id = $1 AND start_time >= ? ORDER BY start_time DESC
CREATE INDEX IF NOT EXISTS idx_trips_vehicle_start_time ON trips(vehicle_id, start_time DESC);

-- 2. Composite index for HasOpenTrip and UpdateTripRecord queries:
--    SELECT COUNT(*) FROM trips WHERE vehicle_id = $1 AND end_time IS NULL
--    SELECT id FROM trips WHERE vehicle_id = $1 AND end_time IS NULL ORDER BY start_time DESC LIMIT 1
CREATE INDEX IF NOT EXISTS idx_trips_vehicle_end_time ON trips(vehicle_id, end_time);

-- Add comments explaining why these indexes are needed
COMMENT ON INDEX idx_trips_vehicle_start_time IS 'Optimizes GetTripHistory: WHERE vehicle_id = ? AND start_time >= ? ORDER BY start_time DESC';
COMMENT ON INDEX idx_trips_vehicle_end_time IS 'Optimizes HasOpenTrip and UpdateTripRecord: WHERE vehicle_id = ? AND end_time IS NULL';

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_vehicle_updated_at 
    BEFORE UPDATE ON vehicle 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_trips_updated_at 
    BEFORE UPDATE ON trips 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample vehicle for testing
INSERT INTO vehicle (id, plate_number) 
VALUES (uuid_generate_v4(), 'ABC-123') 
RETURNING id;

-- Display created tables and sample data
SELECT 'Database schema created successfully!' as status;
SELECT 'Sample vehicle created:' as info;
SELECT id, plate_number, created_at FROM vehicle WHERE plate_number = 'ABC-123';
