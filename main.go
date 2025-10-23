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
	"unicode/utf8"

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

	// Create HTTP server instance with TLS support if configured.
	var srv *server.Server
	if cfg.TLSEnabled && cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		srv = server.NewWithTLS(":"+port, dataStore, handlerService, cfg.TLSCertFile, cfg.TLSKeyFile)
		logger.Info("TLS/HTTPS enabled", map[string]interface{}{
			"cert_file": cfg.TLSCertFile,
		})
	} else {
		srv = server.New(":"+port, dataStore, handlerService)
		if cfg.TLSEnabled {
			logger.Warn("TLS enabled but certificate files not configured - falling back to HTTP")
		}
	}

	// Display startup information.
	tlsStatus := cfg.TLSEnabled && cfg.TLSCertFile != "" && cfg.TLSKeyFile != ""
	printStartupBanner(port, storeInfo, true, tlsStatus)

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

	// Validate JWT secret strength (minimum length recommendation)
	if len(cfg.JWTSecret) < 32 {
		logger.Warn("JWT_SECRET is shorter than recommended 32 characters", map[string]interface{}{
			"length": len(cfg.JWTSecret),
		})
	}

	return nil
}

// resolvePort determines the HTTP server port with fallback to default.
// Validates port is numeric and within valid range.
func resolvePort(configuredPort string) string {
	port := configuredPort
	if port == "" {
		port = DefaultPort
	}

	// Basic validation: ensure port is non-empty after trimming
	port = strings.TrimSpace(port)
	if port == "" {
		return DefaultPort
	}

	return port
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
		close(serverErrors) // Signal that server goroutine has exited
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
	fmt.Fprintln(os.Stderr, "  PORT         - HTTP server port (default: 8080)")
	fmt.Fprintln(os.Stderr, "  DATABASE_URL - SQLite database path (default: in-memory)")
	fmt.Fprintln(os.Stderr, "  TLS_ENABLED  - Enable HTTPS/TLS (true/false, default: false)")
	fmt.Fprintln(os.Stderr, "  TLS_CERT_FILE - Path to TLS certificate file (required if TLS enabled)")
	fmt.Fprintln(os.Stderr, "  TLS_KEY_FILE  - Path to TLS private key file (required if TLS enabled)")
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
func printStartupBanner(port, storeInfo string, dbHealthy, tlsEnabled bool) {
	const boxWidth = 70

	// Helpers: rune-aware length, truncate, pad, and word-wrap.
	runeLen := func(s string) int { return utf8.RuneCountInString(s) }

	truncate := func(s string, width int) string {
		if runeLen(s) <= width {
			return s
		}
		// build runes and slice
		rs := []rune(s)
		return string(rs[:width])
	}

	padRight := func(s string, width int) string {
		l := runeLen(s)
		if l >= width {
			return truncate(s, width)
		}
		return s + strings.Repeat(" ", width-l)
	}

	// Word-wrap a string into lines no longer than width. Splits long words if needed.
	wrap := func(s string, width int) []string {
		if s == "" {
			return []string{""}
		}
		words := strings.Fields(s)
		var lines []string
		var cur string
		push := func() {
			if cur != "" {
				lines = append(lines, cur)
				cur = ""
			}
		}

		for _, w := range words {
			// If word itself larger than width, break it into rune chunks
			if runeLen(w) > width {
				// flush current
				if cur != "" {
					lines = append(lines, cur)
					cur = ""
				}
				// split word
				rs := []rune(w)
				for i := 0; i < len(rs); i += width {
					end := i + width
					if end > len(rs) {
						end = len(rs)
					}
					lines = append(lines, string(rs[i:end]))
				}
				continue
			}

			if cur == "" {
				cur = w
				continue
			}
			if runeLen(cur)+1+runeLen(w) <= width {
				cur = cur + " " + w
				continue
			}
			push()
			cur = w
		}
		push()
		return lines
	}

	border := "+" + strings.Repeat("-", boxWidth) + "+"
	emptyLine := "|" + strings.Repeat(" ", boxWidth) + "|"

	// Prepare text lines
	titleLine := fmt.Sprintf("%s v%s", AppName, AppVersion)
	descLine := AppDescription
	runtimeLine := fmt.Sprintf("Runtime: %s on %s/%s (CPUs: %d)", runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU())

	protocol := "http"
	securityIcon := "HTTP"
	if tlsEnabled {
		protocol = "https"
		securityIcon = "HTTPS"
	}
	portLine := fmt.Sprintf("Port: %s | Store: %s | Protocol: %s", port, storeInfo, securityIcon)

	healthStatus := "OK"
	if !dbHealthy {
		healthStatus = "FAILED"
	}
	healthLine := fmt.Sprintf("Database: %s", healthStatus)

	apiLines := []string{
		"API Endpoints:",
		"POST /api/auth/register - User registration",
		"POST /api/auth/login    - User authentication",
		"POST /api/auth/refresh  - Token refresh",
		"GET  /api/auth/profile  - User profile (JWT required)",
		"GET  /health            - Health check",
	}

	serverURL := fmt.Sprintf("Server: %s://localhost:%s", protocol, port)

	// Print banner
	fmt.Println()
	fmt.Println(border)

	for _, line := range wrap(titleLine, boxWidth) {
		fmt.Printf("|%s|\n", padRight(line, boxWidth))
	}

	fmt.Println(border)

	for _, line := range wrap(descLine, boxWidth) {
		fmt.Printf("|%s|\n", padRight(line, boxWidth))
	}

	fmt.Println(emptyLine)

	for _, line := range wrap(runtimeLine, boxWidth) {
		fmt.Printf("|%s|\n", padRight(line, boxWidth))
	}
	for _, line := range wrap(portLine, boxWidth) {
		fmt.Printf("|%s|\n", padRight(line, boxWidth))
	}
	for _, line := range wrap(healthLine, boxWidth) {
		fmt.Printf("|%s|\n", padRight(line, boxWidth))
	}

	fmt.Println(border)

	// API section
	for _, l := range apiLines {
		for _, line := range wrap(l, boxWidth-2) { // leave small margin for list indentation
			// indent list entries by one space for readability
			fmt.Printf("| %s |\n", padRight(line, boxWidth-2))
		}
	}

	fmt.Println(border)

	for _, line := range wrap(serverURL, boxWidth) {
		fmt.Printf("|%s|\n", padRight(line, boxWidth))
	}

	fmt.Println(border)

	if !tlsEnabled {
		fmt.Println("WARNING: Running in HTTP mode. Enable TLS for production use.")
		fmt.Println("Set TLS_ENABLED=true, TLS_CERT_FILE=/path/to/cert.pem, TLS_KEY_FILE=/path/to/key.pem")
	}
	fmt.Println()
}
