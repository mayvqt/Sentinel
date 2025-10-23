// Package auth provides simple helpers for password hashing and JWT tokens.
// The implementation favors clear, easy-to-read code. Keep Auth instances
// small and pass them to handlers with dependency injection.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mayvqt/Sentinel/internal/config"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrEmptyPassword is returned when callers provide an empty password.
	ErrEmptyPassword = errors.New("empty password")

	// ErrNoSecret is returned when an Auth instance was created without a
	// JWT secret in the configuration.
	ErrNoSecret = errors.New("jwt secret not configured")
)

// Claims is the JWT payload used throughout the API. Keep fields minimal to
// avoid overloading tokens with data.
type Claims struct {
	UserID    string `json:"uid"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// Auth holds the JWT signing secret and provides token helpers.
type Auth struct{ secret string }

// New returns an Auth configured from cfg. If cfg is nil, the returned Auth
// will have an empty secret and token operations will fail with ErrNoSecret.
func New(cfg *config.Config) *Auth {
	var s string
	if cfg != nil {
		s = cfg.JWTSecret
	}
	return &Auth{secret: s}
}

// HashPassword returns a bcrypt hash for pw. Use the returned string for
// storing user passwords. Returns ErrEmptyPassword when pw is empty.
// Uses cost factor of 12 for enhanced security (enterprise-grade).
func HashPassword(pw string) (string, error) {
	if pw == "" {
		return "", ErrEmptyPassword
	}
	// Cost of 12 provides strong security while maintaining reasonable performance
	// Each increment doubles the time, so 12 is ~4x slower than default (10)
	const enterpriseCost = 12
	b, err := bcrypt.GenerateFromPassword([]byte(pw), enterpriseCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword verifies pw against a bcrypt hash. Returns nil on match.
func CheckPassword(hash, pw string) error {
	if hash == "" || pw == "" {
		return bcrypt.ErrMismatchedHashAndPassword
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
}

// GenerateToken signs a JWT for userID with the given role and ttl.
// ttl must be > 0. tokenType should be "access" or "refresh".
func (a *Auth) GenerateToken(userID, role string, ttl time.Duration) (string, error) {
	return a.GenerateTokenWithType(userID, role, "access", ttl)
}

// GenerateTokenWithType signs a JWT with a specific token type.
// tokenType should be "access" or "refresh".
func (a *Auth) GenerateTokenWithType(userID, role, tokenType string, ttl time.Duration) (string, error) {
	if a.secret == "" {
		return "", ErrNoSecret
	}
	if ttl <= 0 {
		return "", errors.New("ttl must be > 0")
	}
	now := time.Now()
	c := Claims{
		UserID:    userID,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString([]byte(a.secret))
}

// ParseToken validates tokenStr and returns its Claims when valid.
func (a *Auth) ParseToken(tokenStr string) (*Claims, error) {
	if a.secret == "" {
		return nil, ErrNoSecret
	}
	if tokenStr == "" {
		return nil, errors.New("token empty")
	}
	c := &Claims{}
	t, err := jwt.ParseWithClaims(tokenStr, c, func(tok *jwt.Token) (interface{}, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, errors.New("token invalid")
	}
	return c, nil
}
