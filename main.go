// Sentinel Authentication Service - Main application entrypoint
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/mayvqt/Sentinel/internal/auth"
	"github.com/mayvqt/Sentinel/internal/config"
	"github.com/mayvqt/Sentinel/internal/handlers"
	"github.com/mayvqt/Sentinel/internal/logger"
	"github.com/mayvqt/Sentinel/internal/server"
	"github.com/mayvqt/Sentinel/internal/store"
)

const (
	appName    = "Sentinel"
	appVersion = "0.1.0"
)

func main() {
	// Print banner
	printBanner()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up logging level
	logger.SetLevel(logger.LevelInfo)

	// Validate required configuration
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Set default port
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	logger.Info("Starting Sentinel Authentication Service", map[string]interface{}{
		"version": appVersion,
		"port":    port,
		"go":      runtime.Version(),
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
	})

	// Initialize store
	var s store.Store
	if cfg.DatabaseURL != "" {
		// Use SQLite store
		s, err = store.NewSQLite(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite store: %v", err)
		}
		logger.Info("Using SQLite store", map[string]interface{}{
			"database_url": cfg.DatabaseURL,
		})
	} else {
		// Fall back to memory store for development
		s = store.NewMemStore()
		logger.Warn("Using in-memory store (development only)")
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
		logger.Info("Starting HTTP server", map[string]interface{}{
			"port": port,
			"addr": ":" + port,
		})

		if err := srv.Start(ctx); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	logger.Info("Received shutdown signal")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		logger.Info("Server shutdown complete")
	}
}

func printBanner() {
	fmt.Println("========================================")
	fmt.Printf(" %s — %s\n", appName, appVersion)
	fmt.Println("----------------------------------------")
	fmt.Printf(" Go version: %s\n", runtime.Version())
	fmt.Printf(" OS/ARCH:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf(" CPUs:        %d\n", runtime.NumCPU())
	fmt.Println("========================================")
}
