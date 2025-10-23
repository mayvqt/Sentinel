// Package store provides persistence interfaces and implementations.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mayvqt/Sentinel/internal/models"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

const (
	// DefaultQueryTimeout is the default timeout for database queries
	DefaultQueryTimeout = 5 * time.Second
	// DefaultTxTimeout is the default timeout for database transactions
	DefaultTxTimeout = 10 * time.Second
)

type sqliteStore struct {
	db *sql.DB
}

// withTimeout creates a context with timeout if one isn't already set
func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		// Context already has a deadline, don't override it
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// NewSQLite opens (or creates) an SQLite database and applies schema.
// It configures WAL, foreign keys, and a tuned connection pool.
func NewSQLite(path string) (Store, error) {
	// Parse database URL to extract path
	dbPath := strings.TrimPrefix(path, "sqlite://")

	// Enterprise SQLite configuration:
	// - _foreign_keys=1: Enable foreign key constraints
	// - _journal_mode=WAL: Write-Ahead Logging for better concurrency
	// - _timeout=5000: 5 second busy timeout
	// - _cache_size=-64000: 64MB cache (negative = KB)
	// - _synchronous=NORMAL: Balance between safety and performance
	db, err := sql.Open("sqlite", dbPath+"?_foreign_keys=1&_journal_mode=WAL&_timeout=5000&_cache_size=-64000&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Configure connection pool for enterprise workloads
	// MaxOpenConns: Maximum number of open connections (25 is good for most workloads)
	// MaxIdleConns: Connections to keep idle (5-10 is typical)
	// ConnMaxLifetime: Maximum time a connection can be reused
	// ConnMaxIdleTime: Maximum time a connection can be idle
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	s := &sqliteStore{db: db}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return s, nil
}

func (s *sqliteStore) init() error {
	// Create users table with proper constraints and indexes
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE COLLATE NOCASE,
		email TEXT UNIQUE COLLATE NOCASE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	
	-- Trigger to update updated_at column
	CREATE TRIGGER IF NOT EXISTS update_users_updated_at 
		AFTER UPDATE ON users
		BEGIN
			UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

func (s *sqliteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *sqliteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *sqliteStore) CreateUser(ctx context.Context, u *models.User) (int64, error) {
	ctx, cancel := withTimeout(ctx, DefaultQueryTimeout)
	defer cancel()

	if u == nil {
		return 0, errors.New("user cannot be nil")
	}
	if u.Username == "" {
		return 0, errors.New("username is required")
	}
	if u.Password == "" {
		return 0, errors.New("password hash is required")
	}
	if u.Role == "" {
		u.Role = "user" // Set default role
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}

	query := `INSERT INTO users (username, email, password_hash, role, created_at) 
			  VALUES (?, ?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query,
		u.Username, u.Email, u.Password, u.Role, u.CreatedAt)
	if err != nil {
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.username") {
			return 0, fmt.Errorf("username '%s' already exists", u.Username)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.email") {
			return 0, fmt.Errorf("email '%s' already exists", u.Email)
		}
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get user ID: %w", err)
	}

	u.ID = id
	return id, nil
}

func (s *sqliteStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	ctx, cancel := withTimeout(ctx, DefaultQueryTimeout)
	defer cancel()

	if username == "" {
		return nil, errors.New("username cannot be empty")
	}

	query := `SELECT id, username, email, password_hash, role, created_at 
			  FROM users WHERE username = ? COLLATE NOCASE`

	row := s.db.QueryRowContext(ctx, query, username)

	u := &models.User{}
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return u, nil
}

func (s *sqliteStore) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	ctx, cancel := withTimeout(ctx, DefaultQueryTimeout)
	defer cancel()

	if id <= 0 {
		return nil, errors.New("user ID must be positive")
	}

	query := `SELECT id, username, email, password_hash, role, created_at 
			  FROM users WHERE id = ?`

	row := s.db.QueryRowContext(ctx, query, id)

	u := &models.User{}
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return u, nil
}
