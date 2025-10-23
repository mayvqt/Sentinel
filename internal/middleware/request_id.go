// Package middleware provides HTTP middleware utilities.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

const (
	// RequestIDKey is the context key for request IDs
	RequestIDKey ContextKey = "request_id"
	// RequestIDHeader is the HTTP header name for request IDs
	RequestIDHeader = "X-Request-ID"
)

// generateRequestID creates a new random request ID.
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a simpler ID if crypto/rand fails
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// WithRequestID adds a unique request ID to each request.
// If the client provides an X-Request-ID header, it will be used;
// otherwise, a new one is generated.
func WithRequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request ID is provided by client
			requestID := r.Header.Get(RequestIDHeader)
			if requestID == "" {
				// Generate new request ID
				requestID = generateRequestID()
			}

			// Add request ID to response header
			w.Header().Set(RequestIDHeader, requestID)

			// Add request ID to context
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

			// Process request with enriched context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestID extracts the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}
