package auth

import (
	"context"
)

// Context key for storing authentication status
type contextKey string

const (
	// AuthenticatedKey indicates if the request is authenticated
	AuthenticatedKey contextKey = "authenticated"
)

// WithAuthentication adds authentication status to the context
func WithAuthentication(ctx context.Context, authenticated bool) context.Context {
	return context.WithValue(ctx, AuthenticatedKey, authenticated)
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(ctx context.Context) bool {
	authenticated, ok := ctx.Value(AuthenticatedKey).(bool)
	return ok && authenticated
}
