// Simple JWT token generator utility
// This script generates JWT tokens using the same secret as the API
// Usage: go run scripts/generate_token.go

package main

import (
	"fmt"
	"os"

	"fleet-tracking/internal/auth"
)

func main() {
	// Delegates secret resolution and signing to internal/auth, the same
	// code the server uses to validate tokens, instead of keeping a second
	// copy here. The second copy used to carry its own hardcoded fallback
	// that didn't match the server's, so a token generated with no
	// JWT_SECRET set would fail to validate against a server started the
	// same way -- exactly the kind of mismatch that makes a quickstart's
	// first request a 401.
	token, err := auth.NewJWTManager().GenerateToken()
	if err != nil {
		fmt.Printf("Error generating token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(token)
	fmt.Fprintf(os.Stderr, "\ncurl -H \"Authorization: Bearer %s\" http://localhost:8080/api/vehicle/status?vehicle_id=<id>\n", token)
}
