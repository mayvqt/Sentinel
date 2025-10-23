// Package store defines persistence interfaces and implementations.
package store

import (
	"context"

	"github.com/mayvqt/Sentinel/internal/models"
)

// Store is the persistence interface used by application services.
// It includes user-focused methods used by the handlers.
type Store interface {
	Close() error
	Ping(ctx context.Context) error

	// CreateUser persists a new user and returns the assigned ID on success.
	CreateUser(ctx context.Context, u *models.User) (int64, error)

	// GetUserByUsername returns a user by username or nil when not found.
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)

	// GetUserByID returns a user by ID.
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
}
