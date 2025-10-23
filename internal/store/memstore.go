package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mayvqt/Sentinel/internal/models"
)

// memStore is a simple in-memory Store implementation for development and
// tests. It is not durable and not intended for production use.
type memStore struct {
	mu     sync.RWMutex
	next   int64
	users  map[int64]*models.User
	byName map[string]int64
}

// NewMemStore constructs a new in-memory store.
func NewMemStore() Store {
	return &memStore{
		next:   1,
		users:  make(map[int64]*models.User),
		byName: make(map[string]int64),
	}
}

func (m *memStore) Close() error { return nil }

func (m *memStore) Ping(ctx context.Context) error { return nil }

func (m *memStore) CreateUser(ctx context.Context, u *models.User) (int64, error) {
	if u == nil {
		return 0, errors.New("nil user")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.next
	m.next++
	u.ID = id
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}
	m.users[id] = u
	m.byName[u.Username] = id
	return id, nil
}

func (m *memStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byName[username]
	if !ok {
		return nil, nil
	}
	u := m.users[id]
	return u, nil
}

func (m *memStore) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u := m.users[id]
	return u, nil
}
