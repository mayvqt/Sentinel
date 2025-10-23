// Package main starts a simple server used to run Sentinel in local mode.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mayvqt/Sentinel/internal/auth"
	"github.com/mayvqt/Sentinel/internal/config"
	"github.com/mayvqt/Sentinel/internal/handlers"
	"github.com/mayvqt/Sentinel/internal/server"
	"github.com/mayvqt/Sentinel/internal/store"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate required configuration
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Set default port
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	// Initialize store
	var s store.Store
	if cfg.DatabaseURL != "" {
		// Use SQLite store
		s, err = store.NewSQLite(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite store: %v", err)
		}
		log.Println("Using SQLite store")
	} else {
		// Fall back to memory store for development
		s = store.NewMemStore()
		log.Println("Using in-memory store (development only)")
	}
	defer s.Close()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize auth and handlers
	a := auth.New(cfg)
	h := handlers.New(s, a)

	// Create and start server
	srv := server.New(":"+port, s, h)

	// Set up graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in a goroutine
	go func() {
		log.Printf("Starting Sentinel server on port %s", port)
		if err := srv.Start(ctx); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	} else {
		log.Println("Server shutdown complete")
	}
}
