// Package config provides typed runtime configuration.
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
	TLSCertFile string
	TLSKeyFile  string
	TLSEnabled  bool
}

// Load reads configuration from .env and environment variables.
func Load() (*Config, error) {
	_ = godotenv.Load()

	locations := []string{".env", ".env.local", "config/.env"}
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
		TLSCertFile: getEnvWithDefault("TLS_CERT_FILE", ""),
		TLSKeyFile:  getEnvWithDefault("TLS_KEY_FILE", ""),
		TLSEnabled:  os.Getenv("TLS_ENABLED") == "true" || os.Getenv("TLS_ENABLED") == "1",
	}, nil
}

// getEnvWithDefault returns the environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
