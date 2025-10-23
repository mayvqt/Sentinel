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

// New constructs a Server. addr is the listen address (":8080").
func New(addr string, s store.Store, h *handlers.Handlers) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/register", h.Register)
	mux.HandleFunc("/login", h.Login)

	// Example of a protected route
	mux.Handle("/me", middleware.WithAuth(h.Auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})))

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return &Server{httpServer: srv, store: s}
}

// Start runs the server until the provided context is canceled.
func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = s.httpServer.Shutdown(context.Background())
	}()
	fmt.Printf("listening on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Close cleans up store and other resources.
func (s *Server) Close() error {
	return s.store.Close()
}
