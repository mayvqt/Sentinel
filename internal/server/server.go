package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mayvqt/Sentinel/internal/handlers"
	"github.com/mayvqt/Sentinel/internal/middleware"
	"github.com/mayvqt/Sentinel/internal/store"
)

// Server holds the components needed to run the HTTP server.
type Server struct {
	httpServer *http.Server
	store      store.Store
}

// New constructs a Server with security middleware and rate limiting.
func New(addr string, s store.Store, h *handlers.Handlers) *Server {
	mux := http.NewServeMux()

	// Create rate limiters for different endpoints
	authRateLimit := middleware.NewRateLimiter(time.Second*2, 5)   // 5 requests per 2 seconds for auth
	generalRateLimit := middleware.NewRateLimiter(time.Second, 10) // 10 requests per second for general

	// Health check endpoint
	mux.Handle("/health", applyMiddleware(
		http.HandlerFunc(h.Health),
		middleware.WithSecurityHeaders(),
		middleware.WithRateLimit(generalRateLimit),
		middleware.WithLogging(),
	))

	// Authentication endpoints with /api/auth prefix and stricter rate limiting
	mux.Handle("/api/auth/register", applyMiddleware(
		http.HandlerFunc(h.Register),
		middleware.WithSecurityHeaders(),
		middleware.WithRateLimit(authRateLimit),
		middleware.WithCORS([]string{"*"}), // Configure allowed origins in production
		middleware.WithLogging(),
	))

	mux.Handle("/api/auth/login", applyMiddleware(
		http.HandlerFunc(h.Login),
		middleware.WithSecurityHeaders(),
		middleware.WithRateLimit(authRateLimit),
		middleware.WithCORS([]string{"*"}), // Configure allowed origins in production
		middleware.WithLogging(),
	))

	// Protected endpoints with /api/auth prefix
	mux.Handle("/api/auth/profile", applyMiddleware(
		http.HandlerFunc(h.Me),
		middleware.WithSecurityHeaders(),
		middleware.WithRateLimit(generalRateLimit),
		middleware.WithCORS([]string{"*"}), // Configure allowed origins in production
		middleware.WithAuth(h.Auth),
		middleware.WithLogging(),
	))

	srv := &http.Server{
		Addr:           addr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return &Server{httpServer: srv, store: s}
}

// applyMiddleware applies middleware in reverse order (last applied runs first).
func applyMiddleware(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// Start runs the server until the provided context is canceled.
func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
	}()

	fmt.Printf("🚀 Sentinel server listening on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Close cleans up store and other resources.
func (s *Server) Close() error {
	return s.store.Close()
}
