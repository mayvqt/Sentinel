// Package config provides typed access to runtime configuration values.
package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
}

// Load returns a Config populated from environment variables and .env files.
// It first attempts to load from .env file, then reads from environment variables.
// Environment variables take precedence over .env file values.
func Load() (*Config, error) {
	// Try to load .env file from current directory
	// This will silently fail if .env doesn't exist, which is fine
	_ = godotenv.Load()

	// Also try to load from common locations
	locations := []string{
		".env",
		".env.local",
		"config/.env",
	}

	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			_ = godotenv.Load(location)
			break
		}
	}

	return &Config{
		Port:        getEnvWithDefault("PORT", ""),
		DatabaseURL: getEnvWithDefault("DATABASE_URL", ""),
		JWTSecret:   getEnvWithDefault("JWT_SECRET", ""),
	}, nil
}

// getEnvWithDefault returns the environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
