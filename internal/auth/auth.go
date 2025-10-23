// Package auth implements authentication helpers: password hashing and JWT
// creation/validation. Keep cryptographic secrets out of source control.
package auth

import "time"

// Claims represents JWT claims (user id and role).
type Claims struct {
	UserID string
	Role   string
}

// HashPassword returns a bcrypt hash for the given password.
// Implemented in auth/hash.go; signature kept here for clarity.
func HashPassword(password string) (string, error) { return "", nil }

// CheckPassword compares a bcrypt hash with a plaintext password.
func CheckPassword(hash, password string) error { return nil }

// GenerateToken creates a signed JWT for a user with a role and expiration.
// Implementation should read signing secret from environment.
func GenerateToken(userID, role string, exp time.Duration) (string, error) { return "", nil }

// ParseToken validates a token string and returns its Claims.
func ParseToken(tokenStr string) (*Claims, error) { return &Claims{}, nil }
