// Package main starts the Sentinel authentication service.
// It wires configuration, logging, storage, auth, and HTTP server.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
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

// main is the application entrypoint. It initializes services and runs the server.
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

// validateConfiguration checks required config values.
func validateConfiguration(cfg *config.Config) error {
	if cfg.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}
	return nil
}

// printConfigurationHelp prints setup tips when configuration is missing.
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

// logSystemInfo logs basic runtime and system information.
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

// initializeStore returns the configured Store implementation.
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

// testDatabaseConnection pings the store to verify connectivity.
func testDatabaseConnection(s store.Store) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	logger.Info("✅ Database connection verified")
	return nil
}

// runServerWithGracefulShutdown runs the server and performs graceful shutdown.
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

		// Use consistent boxed output for clean alignment
		fmt.Println()
		printBoxTop()
		printBoxCenterf("🚀 Sentinel server listening on :%s", port)
		printBoxEmpty()
		printBoxLeftf("📍 API Base URL: http://localhost:%s/api", port)
		printBoxLeft("📖 Endpoints:")
		printBoxLeft("POST /api/auth/register - Create new user account")
		printBoxLeft("POST /api/auth/login    - Authenticate existing user")
		printBoxLeft("GET  /api/auth/profile  - Get user profile (requires JWT)")
		printBoxLeft("GET  /health            - Service health check")
		printBoxEmpty()
		printBoxLeft("💡 Press Ctrl+C to gracefully shutdown")
		printBoxBottom()
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
	printBoxTop()
	printBoxCenterf("🛡️ %s v%s", AppName, AppVersion)
	printBoxEmpty()
	printBoxLeft(AppDescription)
	printBoxEmpty()
	printBoxLeftf("🔧 Runtime: %s  Platform: %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	printBoxLeftf("⚡ CPUs: %d  Author: %s", runtime.NumCPU(), AppAuthor)
	printBoxBottom()
	fmt.Println()
}

// Box printing helpers
const boxWidth = 71 // inner width of the box

func padToWidth(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func printBoxTop() {
	fmt.Println("╔" + strings.Repeat("═", boxWidth) + "╗")
}

func printBoxBottom() {
	fmt.Println("╚" + strings.Repeat("═", boxWidth) + "╝")
}

func printBoxEmpty() {
	fmt.Printf("║%s║\n", padToWidth("", boxWidth))
}

func printBoxLeft(s string) {
	fmt.Printf("║ %s %s║\n", s, padToWidth("", boxWidth-2-len(s)))
}

func printBoxLeftf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	if len(s) > boxWidth-2 {
		s = s[:boxWidth-5] + "..."
	}
	printBoxLeft(s)
}

func printBoxCenterf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	if len(s) > boxWidth {
		s = s[:boxWidth]
	}
	left := (boxWidth - len(s)) / 2
	right := boxWidth - len(s) - left
	fmt.Printf("║%s%s%s║\n", strings.Repeat(" ", left), s, strings.Repeat(" ", right))
}
