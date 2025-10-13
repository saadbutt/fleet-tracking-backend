# Fleet Tracking Backend Service
## Quick Start

### Prerequisites

- **Docker & Docker Compose** 
- **Go 1.25+**
- **PostgreSQL 15+**
- **Redis 7+** 

### Option 1: Docker Compose

```bash
# Clone the repository
git clone https://github.com/saadbutt/fleet-tracking-backend
cd fleet-tracking-backend

# Start all services
docker-compose up -d

# Run migration to populate data
docker exec -i fleet-tracking-postgres psql -U postgres -d fleet_tracking < migrations/001_complete_schema.sql

# Get vehicle Id to use in APIs
docker exec -it fleet-tracking-postgres psql -U postgres -d fleet_tracking -c "SELECT id, plate_number, created_at FROM vehicle;"

# Generate JWT token
go run scripts/generate_token.go

```

The API will be available at: `http://localhost:8080` and Swagger will be available at: http://localhost:8080/swagger

### Option 2: Local Development

```bash

go mod tidy

# Set up environment variables
cp .env.example .env

# Populate .env with required configurations

createdb fleet_tracking -U postgres

psql -h localhost -U postgres -d fleet_tracking -f migrations/001_complete_schema.sql

# Note: The sensor simulation will automatically start for the first vehicle in the database

# Run the application
go run cmd/api/main.go
```

## Run TestCase

```bash
# Run all unit tests
go test ./internal/... -v
```

## API Documentation

### Base URL
```
http://localhost:8080
```

### Authentication
All API endpoints require JWT authentication. Include the token in the Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

### Generate Test Token
```bash
go run scripts/generate_token.go > token.txt
curl -H "Authorization: Bearer $(cat token.txt)" ...
```

### Swagger Documentation
Visit `http://localhost:8080/swagger/` for interactive API documentation.

## API Endpoints

### 1. Health Check

```bash
# Check service health
curl -X GET http://localhost:8080/health
```

### 2. Get Vehicle Status

```bash
# Get current vehicle status
curl -X GET "http://localhost:8080/api/vehicle/status?vehicle_id=:vehicleId" \
  -H "Authorization: Bearer <your-jwt-token>"
```

**Response:**
```json
{
  "vehicle_id": ":vehicleId",
  "plate_number": "ABC-123",
  "last_status": {
    "location": [25.2048, 55.2708],
    "speed": 60.5,
    "timestamp": "2025-10-13T10:00:00Z"
  },
  "last_updated": "2025-10-13T10:00:00Z"
}
```

### 3. Get Trip History

```bash
# Get trip history for past 24 hours
curl -X GET "http://localhost:8080/api/vehicle/trips?vehicle_id=:vehicleId&hours=2" \
  -H "Authorization: Bearer <your-jwt-token>"
```

**Response:**
```json
{
  "trips": [
    {
      "id": "trip-uuid-1",
      "vehicle_id": ":vehicleId",
      "start_time": "2025-10-13T09:00:00Z",
      "end_time": "2025-10-13T09:30:00Z",
      "mileage": 15.5,
      "avg_speed": 45.2
    }
  ],
  "count": 1
}
```

### 4. Ingest Vehicle Data

```bash
# Ingest new vehicle data
curl -X POST http://localhost:8080/api/vehicle/ingest \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -d '{
    "vehicle_id": ":vehicleId",
    "status": {
      "location": [25.2048, 55.2708],
      "speed": 60.5,
      "timestamp": "2025-10-13T10:00:00Z"
    }
  }'
```

**Response:**
```json
{
  "message": "Vehicle data ingested successfully",
  "vehicle_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## API Testing

### Swagger UI
The easiest way to test the API is using the built-in Swagger UI:
1. Start the services: `docker-compose up -d`
2. Open your browser: `http://localhost:8080/swagger/`
3. Use the "Authorize" button to set your JWT token
4. Test all endpoints directly in the browser

For more detail use this README: [`README/`](./docs/README.md)
