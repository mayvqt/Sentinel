package middleware

import (
	"context"
	"net/http"

	"github.com/mayvqt/Sentinel/internal/auth"
)

// WithAuth returns a middleware that validates the Bearer token and stores
// the parsed claims in the request context.
func WithAuth(a *auth.Auth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Expect format: "Bearer <token>"
			const bearerPrefix = "Bearer "
			if len(authHeader) <= len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
				writeAuthError(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := authHeader[len(bearerPrefix):]
			claims, err := a.ParseToken(token)
			if err != nil {
				writeAuthError(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Add claims to request context
			ctx := context.WithValue(r.Context(), "user", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// writeAuthError writes a structured authentication error response.
func writeAuthError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error":"` + http.StatusText(statusCode) + `","message":"` + message + `"}`))
}
