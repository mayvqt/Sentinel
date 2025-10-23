/*
███████╗███████╗███╗   ██╗████████╗██╗███╗   ██╗███████╗██╗
██╔════╝██╔════╝████╗  ██║╚══██╔══╝██║████╗  ██║██╔════╝██║
███████╗█████╗  ██╔██╗ ██║   ██║   ██║██╔██╗ ██║█████╗  ██║
╚════██║██╔══╝  ██║╚██╗██║   ██║   ██║██║╚██╗██║██╔══╝  ██║
███████║███████╗██║ ╚████║   ██║   ██║██║ ╚████║███████╗███████╗
╚══════╝╚══════╝╚═╝  ╚═══╝   ╚═╝   ╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝

Sentinel Authentication Service
Enterprise-grade JWT authentication microservice built with Go
*/
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

// Application metadata
const (
	AppName        = "Sentinel"
	AppVersion     = "0.1.0"
	AppDescription = "Enterprise-grade JWT authentication microservice"
	AppAuthor      = "mayvqt"
)

// main is the application entrypoint. It orchestrates service initialization,
// configuration loading, and graceful shutdown handling.
func main() {
	// ┌─────────────────────────────────────────────────────────────────────┐
	// │                          INITIALIZATION                             │
	// └─────────────────────────────────────────────────────────────────────┘

	// Display application banner
	printBanner()

	// Load configuration from environment and .env files
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Configuration loading failed: %v", err)
	}

	// Initialize structured logging
	logger.SetLevel(logger.LevelInfo)
	logger.Info("Configuration loaded successfully", map[string]interface{}{
		"app":     AppName,
		"version": AppVersion,
	})

	// ┌─────────────────────────────────────────────────────────────────────┐
	// │                          VALIDATION                                 │
	// └─────────────────────────────────────────────────────────────────────┘

	if err := validateConfiguration(cfg); err != nil {
		printConfigurationHelp()
		os.Exit(1)
	}

	// Set default port if not specified
	port := cfg.Port
	if port == "" {
		port = "8080"
		logger.Info("Using default port", map[string]interface{}{"port": port})
	}

	// ┌─────────────────────────────────────────────────────────────────────┐
	// │                       SERVICE STARTUP                               │
	// └─────────────────────────────────────────────────────────────────────┘

	logSystemInfo(port)

	// Initialize data store
	store, err := initializeStore(cfg)
	if err != nil {
		log.Fatalf("❌ Store initialization failed: %v", err)
	}
	defer store.Close()

	// Test database connectivity
	if err := testDatabaseConnection(store); err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}

	// Initialize authentication and handlers
	authService := auth.New(cfg)
	handlerService := handlers.New(store, authService)

	// Create HTTP server
	srv := server.New(":"+port, store, handlerService)

	// ┌─────────────────────────────────────────────────────────────────────┐
	// │                      GRACEFUL SHUTDOWN                              │
	// └─────────────────────────────────────────────────────────────────────┘

	runServerWithGracefulShutdown(srv, port)
}

// ┌─────────────────────────────────────────────────────────────────────────┐
// │                            HELPER FUNCTIONS                             │
// └─────────────────────────────────────────────────────────────────────────┘

// validateConfiguration ensures all required configuration values are present
func validateConfiguration(cfg *config.Config) error {
	if cfg.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}
	return nil
}

// printConfigurationHelp displays helpful setup instructions for missing config
func printConfigurationHelp() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         ⚠️  CONFIGURATION ERROR                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("❌ Missing required environment variable: JWT_SECRET")
	fmt.Println()
	fmt.Println("🔧 Quick Setup (PowerShell):")
	fmt.Println("   $env:JWT_SECRET = \"your-secure-secret-key\"")
	fmt.Println("   $env:PORT = \"8080\"")
	fmt.Println("   $env:DATABASE_URL = \"sqlite://./sentinel.db\"")
	fmt.Println("   go run .")
	fmt.Println()
	fmt.Println("📝 Alternative - Create .env file:")
	fmt.Println("   JWT_SECRET=your-secure-secret-key")
	fmt.Println("   PORT=8080")
	fmt.Println("   DATABASE_URL=sqlite://./sentinel.db")
	fmt.Println()
	fmt.Println("🔒 Generate a secure JWT secret:")
	fmt.Println("   [System.Security.Cryptography.RNGCryptoServiceProvider]::Create().GetBytes(32) | ForEach {$_.ToString('x2')} | Join-String")
	fmt.Println()
}

// logSystemInfo displays system and runtime information
func logSystemInfo(port string) {
	logger.Info("🚀 Initializing Sentinel Authentication Service", map[string]interface{}{
		"app":     AppName,
		"version": AppVersion,
		"port":    port,
		"go":      runtime.Version(),
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
		"cpus":    runtime.NumCPU(),
	})
}

// initializeStore creates and configures the data storage backend
func initializeStore(cfg *config.Config) (store.Store, error) {
	var s store.Store
	var err error

	if cfg.DatabaseURL != "" {
		// Production SQLite store
		s, err = store.NewSQLite(cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("SQLite store initialization failed: %w", err)
		}
		logger.Info("✅ SQLite store initialized", map[string]interface{}{
			"database_url": cfg.DatabaseURL,
		})
	} else {
		// Development in-memory store
		s = store.NewMemStore()
		logger.Warn("⚠️  Using in-memory store (development only - data will not persist)")
	}

	return s, nil
}

// testDatabaseConnection verifies database connectivity
func testDatabaseConnection(s store.Store) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	logger.Info("✅ Database connection verified")
	return nil
}

// runServerWithGracefulShutdown starts the HTTP server and handles graceful shutdown
func runServerWithGracefulShutdown(srv *server.Server, port string) {
	// Set up graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in background goroutine
	go func() {
		logger.Info("🌐 HTTP server starting", map[string]interface{}{
			"port":    port,
			"address": ":" + port,
			"url":     fmt.Sprintf("http://localhost:%s", port),
		})

		fmt.Println()
		fmt.Println("╔═══════════════════════════════════════════════════════════════════════╗")
		fmt.Printf("║                     🚀 Sentinel server listening on :%s               ║\n", port)
		fmt.Println("║                                                                       ║")
		fmt.Printf("║  📍 API Base URL: http://localhost:%s/api                           ║\n", port)
		fmt.Println("║  📖 Endpoints:                                                        ║")
		fmt.Println("║     POST /api/auth/register - Create new user account                ║")
		fmt.Println("║     POST /api/auth/login    - Authenticate existing user             ║")
		fmt.Println("║     GET  /api/auth/profile  - Get user profile (requires JWT)        ║")
		fmt.Println("║     POST /api/auth/refresh  - Refresh JWT token                      ║")
		fmt.Println("║                                                                       ║")
		fmt.Println("║  💡 Press Ctrl+C to gracefully shutdown                              ║")
		fmt.Println("╚═══════════════════════════════════════════════════════════════════════╝")
		fmt.Println()

		if err := srv.Start(ctx); err != nil && err != http.ErrServerClosed {
			logger.Error("❌ Server startup failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("🛑 Shutdown signal received - initiating graceful shutdown")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("❌ Server shutdown error", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		logger.Info("✅ Server shutdown completed successfully")
		fmt.Println()
		fmt.Println("👋 Thank you for using Sentinel! Have a great day!")
		fmt.Println()
	}
}

// printBanner displays a professional application banner with system information
func printBanner() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║                          🛡️  %s v%s                          ║\n", AppName, AppVersion)
	fmt.Println("║                                                                       ║")
	fmt.Printf("║  %s  ║\n", AppDescription)
	fmt.Println("║                                                                       ║")
	fmt.Printf("║  🔧 Runtime: %-20s 💻 Platform: %s/%s     ║\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("║  ⚡ CPUs:    %-20d 👤 Author:   %-15s ║\n", runtime.NumCPU(), AppAuthor)
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}
