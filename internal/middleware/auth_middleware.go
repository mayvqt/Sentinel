package middleware

import (
	"context"
	"net/http"

	"github.com/mayvqt/Sentinel/internal/auth"
)

// key type for context
type ctxKey string

const UserCtxKey ctxKey = "user"

// WithAuth returns a middleware that validates the Bearer token and stores
// the parsed claims in the request context under UserCtxKey.
func WithAuth(a *auth.Auth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hdr := r.Header.Get("Authorization")
			if hdr == "" {
				http.Error(w, "missing authorization", http.StatusUnauthorized)
				return
			}
			// Expect format: "Bearer <token>"
			const prefix = "Bearer "
			if len(hdr) <= len(prefix) || hdr[:len(prefix)] != prefix {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}
			token := hdr[len(prefix):]
			claims, err := a.ParseToken(token)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), UserCtxKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
