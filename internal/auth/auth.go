// Package auth provides password hashing and JWT helpers.
// It supports access and refresh tokens used by the API.
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

// Claims is the JWT payload used throughout the API.
// Keep fields minimal to avoid overloading tokens with data.
type Claims struct {
	UserID    string `json:"uid"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

type Auth struct{ secret string }

// New returns an Auth configured from cfg. If cfg is nil, operations will fail.
func New(cfg *config.Config) *Auth {
	var s string
	if cfg != nil {
		s = cfg.JWTSecret
	}
	return &Auth{secret: s}
}

// HashPassword returns a bcrypt hash for pw. Returns ErrEmptyPassword if pw is empty.
// Uses cost factor 12 for strong security.
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

// CheckPassword compares a bcrypt hash with the provided password.
func CheckPassword(hash, pw string) error {
	if hash == "" || pw == "" {
		return bcrypt.ErrMismatchedHashAndPassword
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
}

// GenerateToken signs an access JWT for userID with the given role and ttl.
func (a *Auth) GenerateToken(userID, role string, ttl time.Duration) (string, error) {
	return a.GenerateTokenWithType(userID, role, "access", ttl)
}

// GenerateTokenWithType signs a JWT with a specific tokenType ("access" or "refresh").
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

	// Explicit expiry check (jwt library checks this, but we add explicit validation)
	if c.ExpiresAt != nil && time.Now().After(c.ExpiresAt.Time) {
		return nil, errors.New("token expired")
	}

	// Validate issued-at time is not in the future (clock skew tolerance: 1 minute)
	// This prevents tokens with IssuedAt far in the future while allowing minor clock drift
	if c.IssuedAt != nil {
		now := time.Now()
		maxFutureSkew := 1 * time.Minute
		if c.IssuedAt.Time.After(now.Add(maxFutureSkew)) {
			return nil, errors.New("token issued too far in the future")
		}
	}

	return c, nil
}
