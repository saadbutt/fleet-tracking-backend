// Simple JWT token generator utility
// This script generates JWT tokens using the same secret as the API
// Usage: go run scripts/generate_token.go

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	// Simple authentication - just indicates the request is authorized
	Authorized bool `json:"authorized"`
	jwt.RegisteredClaims
}

func main() {
	// Get secret from environment or use default
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "fleet-tracking-secret-key-2024-production"
	}

	// Create claims
	claims := JWTClaims{
		Authorized: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "fleet-tracking-api",
			Subject:   "fleet-tracking-client",
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		fmt.Printf("Error generating token: %v\n", err)
		os.Exit(1)
	}

	// Output token information
	response := map[string]interface{}{
		"token":      tokenString,
		"expires_in": 86400, // 24 hours in seconds
		"token_type": "Bearer",
		"authorized": claims.Authorized,
		"expires_at": claims.ExpiresAt,
	}

	jsonOutput, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated JWT Token:")
	fmt.Println(string(jsonOutput))
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:8080/api/vehicle/status?vehicle_id=53d0ad1f-3fc6-4b96-9252-faf421e98117\n", tokenString)
}
