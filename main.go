// Package main is the entry point for the Sentinel authentication service.
// It orchestrates configuration loading, dependency initialization, and HTTP server lifecycle.
package main

import (
	"context"
	"errors"
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

// Application metadata constants.
const (
	AppName        = "Sentinel"
	AppVersion     = "0.1.0"
	AppDescription = "Enterprise-grade JWT authentication microservice"
	AppAuthor      = "mayvqt"
)

// Exit codes for different failure scenarios.
const (
	ExitCodeSuccess         = 0
	ExitCodeConfigError     = 1
	ExitCodeStoreError      = 2
	ExitCodeServerError     = 3
	ExitCodeShutdownTimeout = 4
)

// Operational timeouts.
const (
	DatabasePingTimeout     = 5 * time.Second
	GracefulShutdownTimeout = 30 * time.Second
	DefaultPort             = "8080"
)

func main() {
	os.Exit(run())
}

// run encapsulates the main application logic and returns an exit code.
// This pattern enables proper cleanup via deferred functions and testability.
func run() int {
	// Initialize structured logging subsystem.
	logger.SetLevel(logger.LevelInfo)

	// Load configuration from environment and .env file.
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Configuration load failed: %v", err)
		return ExitCodeConfigError
	}

	// Validate required configuration parameters.
	if err := validateConfiguration(cfg); err != nil {
		printConfigurationHelp(err)
		return ExitCodeConfigError
	}

	// Determine server port with fallback to default.
	port := resolvePort(cfg.Port)

	// Initialize data store (SQLite or in-memory).
	dataStore, storeInfo, err := initializeStore(cfg)
	if err != nil {
		log.Printf("Store initialization failed: %v", err)
		return ExitCodeStoreError
	}
	defer func() {
		if closeErr := dataStore.Close(); closeErr != nil {
			logger.Error("Store cleanup failed", map[string]interface{}{
				"error": closeErr.Error(),
			})
		}
	}()

	// Verify database connectivity before proceeding.
	ctx, cancel := context.WithTimeout(context.Background(), DatabasePingTimeout)
	defer cancel()

	if err := dataStore.Ping(ctx); err != nil {
		log.Printf("Database connectivity check failed: %v", err)
		return ExitCodeStoreError
	}

	// Initialize authentication service.
	authService := auth.New(cfg)

	// Initialize HTTP handlers.
	handlerService := handlers.New(dataStore, authService)

	// Create HTTP server instance.
	srv := server.New(":"+port, dataStore, handlerService)

	// Display startup information.
	printStartupBanner(port, storeInfo, true)

	// Run server with graceful shutdown handling.
	if err := runServerWithGracefulShutdown(srv); err != nil {
		log.Printf("Server execution failed: %v", err)
		return ExitCodeServerError
	}

	logger.Info("Service terminated successfully")
	return ExitCodeSuccess
}

// validateConfiguration validates all required configuration parameters.
func validateConfiguration(cfg *config.Config) error {
	if cfg == nil {
		return errors.New("configuration is nil")
	}

	if cfg.JWTSecret == "" {
		return errors.New("JWT_SECRET is required")
	}

	return nil
}

// resolvePort determines the HTTP server port with fallback to default.
func resolvePort(configuredPort string) string {
	if configuredPort != "" {
		return configuredPort
	}
	return DefaultPort
}

// initializeStore creates and configures the data store based on configuration.
func initializeStore(cfg *config.Config) (store.Store, string, error) {
	if cfg.DatabaseURL != "" {
		// Production mode: use SQLite persistent store.
		sqlStore, err := store.NewSQLite(cfg.DatabaseURL)
		if err != nil {
			return nil, "", fmt.Errorf("SQLite initialization: %w", err)
		}
		storeDesc := fmt.Sprintf("SQLite (%s)", cfg.DatabaseURL)
		return sqlStore, storeDesc, nil
	}

	// Development mode: use in-memory ephemeral store.
	memStore := store.NewMemStore()
	logger.Warn("Using in-memory store (data will not persist across restarts)")
	return memStore, "in-memory (development)", nil
}

// runServerWithGracefulShutdown starts the HTTP server and handles shutdown signals.
func runServerWithGracefulShutdown(srv *server.Server) error {
	// Create context that cancels on interrupt or termination signal.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	// Channel to capture server startup errors.
	serverErrors := make(chan error, 1)

	// Start HTTP server in background goroutine.
	go func() {
		if err := srv.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("server start: %w", err)
		}
	}()

	// Block until shutdown signal or server error.
	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		logger.Info("Shutdown signal received")
	}

	// Initiate graceful shutdown with timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		GracefulShutdownTimeout,
	)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("Server shutdown completed")
	return nil
}

// printConfigurationHelp displays setup instructions when configuration is invalid.
func printConfigurationHelp(validationErr error) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Configuration Error: %v\n", validationErr)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Required Configuration:")
	fmt.Fprintln(os.Stderr, "  JWT_SECRET - Secret key for JWT token signing")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Optional Configuration:")
	fmt.Fprintln(os.Stderr, "  PORT        - HTTP server port (default: 8080)")
	fmt.Fprintln(os.Stderr, "  DATABASE_URL - SQLite database path (default: in-memory)")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Setup Methods:")
	fmt.Fprintln(os.Stderr, "  1. Environment variables")
	fmt.Fprintln(os.Stderr, "  2. .env file in project root")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Generate Secure Secret:")
	fmt.Fprintln(os.Stderr, "  openssl rand -base64 32")
	fmt.Fprintln(os.Stderr, "  (PowerShell): [Convert]::ToBase64String([byte[]](1..32|%{Get-Random -Max 256}))")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "See README.md for detailed documentation")
	fmt.Fprintln(os.Stderr)
}

// printStartupBanner displays service information and available endpoints.
func printStartupBanner(port, storeInfo string, dbHealthy bool) {
	const boxWidth = 70

	// Safe padding helper that prevents negative repeat counts.
	pad := func(s string, width int) string {
		if len(s) >= width {
			return s
		}
		padding := width - len(s)
		if padding < 0 {
			padding = 0
		}
		return s + strings.Repeat(" ", padding)
	}

	border := "+" + strings.Repeat("-", boxWidth) + "+"
	emptyLine := "|" + strings.Repeat(" ", boxWidth) + "|"

	// Build formatted lines with safe padding.
	titleLine := fmt.Sprintf(" %s v%s ", AppName, AppVersion)
	descLine := fmt.Sprintf(" %s ", AppDescription)
	runtimeLine := fmt.Sprintf(" Runtime: %s on %s/%s (CPUs: %d) ",
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	portLine := fmt.Sprintf(" Port: %s | Store: %s ", port, storeInfo)

	healthStatus := "OK"
	if !dbHealthy {
		healthStatus = "FAILED"
	}
	healthLine := fmt.Sprintf(" Database: %s ", healthStatus)

	// Print banner.
	fmt.Println()
	fmt.Println(border)
	fmt.Printf("|%s|\n", pad(titleLine, boxWidth))
	fmt.Println(border)
	fmt.Printf("|%s|\n", pad(descLine, boxWidth))
	fmt.Println(emptyLine)
	fmt.Printf("|%s|\n", pad(runtimeLine, boxWidth))
	fmt.Printf("|%s|\n", pad(portLine, boxWidth))
	fmt.Printf("|%s|\n", pad(healthLine, boxWidth))
	fmt.Println(border)
	fmt.Printf("| %-*s |\n", boxWidth-2, "API Endpoints:")
	fmt.Printf("| %-*s |\n", boxWidth-2, "  POST /api/auth/register - User registration")
	fmt.Printf("| %-*s |\n", boxWidth-2, "  POST /api/auth/login    - User authentication")
	fmt.Printf("| %-*s |\n", boxWidth-2, "  POST /api/auth/refresh  - Token refresh")
	fmt.Printf("| %-*s |\n", boxWidth-2, "  GET  /api/auth/profile  - User profile (JWT required)")
	fmt.Printf("| %-*s |\n", boxWidth-2, "  GET  /health            - Health check")
	fmt.Println(border)
	serverURL := fmt.Sprintf(" Server: http://localhost:%s ", port)
	fmt.Printf("|%s|\n", pad(serverURL, boxWidth))
	fmt.Println(border)
	fmt.Println()
}
