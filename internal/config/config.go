// Package config provides typed access to runtime configuration values.
package config

import "os"

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
}

// Load returns a Config populated from environment variables. Validation and
// defaults are intentionally left to the caller.
func Load() (*Config, error) {
	return &Config{
		Port:        os.Getenv("PORT"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}, nil
}
