package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mayvqt/Sentinel/internal/models"
)

type sqliteStore struct {
	db *sql.DB
}

// NewSQLite creates or opens an SQLite database at path.
func NewSQLite(path string) (Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	s := &sqliteStore{db: db}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *sqliteStore) init() error {
	schema := `CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL UNIQUE,
        email TEXT,
        password_hash TEXT NOT NULL,
        role TEXT,
        created_at DATETIME
    );`
	_, err := s.db.Exec(schema)
	return err
}

func (s *sqliteStore) Close() error { return s.db.Close() }

func (s *sqliteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *sqliteStore) CreateUser(ctx context.Context, u *models.User) (int64, error) {
	if u == nil {
		return 0, errors.New("nil user")
	}
	if u.Username == "" || u.Password == "" {
		return 0, errors.New("username/password required")
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}

	res, err := s.db.ExecContext(ctx, `INSERT INTO users (username,email,password_hash,role,created_at) VALUES (?,?,?,?,?)`,
		u.Username, u.Email, u.Password, u.Role, u.CreatedAt)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	u.ID = id
	return id, nil
}

func (s *sqliteStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,username,email,password_hash,role,created_at FROM users WHERE username = ?`, username)
	u := &models.User{}
	var created string
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
		u.CreatedAt = t
	}
	return u, nil
}

func (s *sqliteStore) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,username,email,password_hash,role,created_at FROM users WHERE id = ?`, id)
	u := &models.User{}
	var created string
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
		u.CreatedAt = t
	}
	return u, nil
}
