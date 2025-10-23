// Package store defines persistence interfaces and implementations.
package store

import "context"

// Store is the persistence interface used by application services.
type Store interface {
	Close() error
	Ping(ctx context.Context) error
}
