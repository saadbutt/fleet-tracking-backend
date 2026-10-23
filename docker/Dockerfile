# Build stage
FROM golang:1.26-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install swag for generating Swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger documentation
RUN swag init -g cmd/api/main.go

# Fix compatibility issue with swaggo/swag - remove unsupported fields
RUN sed -i '/LeftDelim/d; /RightDelim/d' docs/docs.go

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Final stage
FROM alpine:latest

# Install ca-certificates and curl for health checks
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy migrations directory
COPY --from=builder /app/migrations ./migrations

# Change ownership to appuser
RUN chown -R appuser:appuser /root/

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"]
