// @title Fleet Tracking API
// @version 1.0
// @description A simple fleet tracking API for vehicle management
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"bufio"
	"context"
	"database/sql"
	"fleet-tracking/docs"
	"fleet-tracking/internal/config"
	"fleet-tracking/internal/controllers"
	"fleet-tracking/internal/middleware"
	"fleet-tracking/internal/repository"
	"fleet-tracking/internal/service"
	"fleet-tracking/pkg/logger"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	// Initialize Swagger docs
	docs.SwaggerInfo.Title = "Fleet Tracking API"
	docs.SwaggerInfo.Description = "A simple fleet tracking API for vehicle management"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:8080"
	docs.SwaggerInfo.BasePath = "/"

	// Load environment variables from .env file
	loadEnvFile(".env")

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/fleet_tracking?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Error("Failed to connect to database: ", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database: ", err)
		os.Exit(1)
	}
	logger.Info("Database connection established")

	// Initialize Redis connection
	redisConfig := config.LoadRedisConfig()
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Host + ":" + redisConfig.Port,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})
	defer rdb.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect to Redis: ", err)
		os.Exit(1)
	}
	logger.Info("Redis connection established")

	// Initialize dependencies
	healthRepo := repository.NewHealthRepository()
	healthService := service.NewHealthService(healthRepo)
	healthController := controllers.NewHealthController(healthService)

	vehicleRepo := repository.NewVehicleRepository(db)
	cacheService := service.NewCacheService(rdb)
	vehicleService := service.NewVehicleService(vehicleRepo, cacheService)
	simulatorService := service.NewSimulatorService(vehicleService)
	vehicleController := controllers.NewVehicleController(vehicleService)

	// Initialize and start the sensor simulator
	go func() {
		logger.Info("Starting sensor simulation service...")

		// Wait for database to be ready and vehicles to be seeded
		for {
			vehicleID, err := vehicleRepo.GetFirstVehicleID()
			if err != nil {
				logger.Info("Waiting for database to be seeded with vehicles... (run 'make seed' to create tables and sample data)")
				time.Sleep(10 * time.Second)
				continue
			}

			// Start simulation for the first available vehicle
			simulatorService.StartSimulation(ctx, vehicleID)
			logger.Info("Started automatic sensor simulation for vehicle: " + vehicleID)
			logger.Info("Sensor data will be generated every 2 seconds")
			return
		}
	}()

	// Create HTTP mux and setup routes
	mux := http.NewServeMux()
	setupRoutes(mux, healthController, vehicleController)

	logger.Info("Starting Fleet Tracking API server...")
	logger.Info("Starting server on port: ", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Error("Failed to start server: ", err)
		os.Exit(1)
	}
}

// setupRoutes configures all the routes
func setupRoutes(mux *http.ServeMux, healthController *controllers.HealthController, vehicleController *controllers.VehicleController) {
	// Initialize error handler
	errorHandler := middleware.NewErrorHandler()

	// Swagger documentation
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// Health check endpoint
	mux.HandleFunc("/health", middleware.Chain(
		healthController.HealthCheck,
		middleware.ObservabilityMiddleware(),
		middleware.RecoverMiddleware(errorHandler),
	))

	// API v1 routes
	mux.HandleFunc("/api/v1/health", middleware.Chain(
		healthController.HealthCheck,
		middleware.ObservabilityMiddleware(),
		middleware.RecoverMiddleware(errorHandler),
	))

	// Vehicle endpoints - protected with JWT authentication
	mux.HandleFunc("/api/vehicle/status", middleware.Chain(
		vehicleController.GetVehicleStatus,
		middleware.JWT(),
		middleware.ObservabilityMiddleware(),
		middleware.RecoverMiddleware(errorHandler),
	))

	mux.HandleFunc("/api/vehicle/trips", middleware.Chain(
		vehicleController.GetTripHistory,
		middleware.JWT(),
		middleware.ObservabilityMiddleware(),
		middleware.RecoverMiddleware(errorHandler),
	))

	// Vehicle ingest endpoint - protected with JWT authentication
	mux.HandleFunc("/api/vehicle/ingest", middleware.Chain(
		vehicleController.IngestVehicleData,
		middleware.JWT(),
		middleware.ObservabilityMiddleware(),
		middleware.RecoverMiddleware(errorHandler),
	))

}

// loadEnvFile loads environment variables from an .env file
func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}
