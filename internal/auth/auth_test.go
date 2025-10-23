package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mayvqt/Sentinel/internal/config"
)

func TestHashAndCheckPassword(t *testing.T) {
	pw := "correct-horse-battery-staple"
	h, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if h == "" {
		t.Fatalf("expected non-empty hash")
	}
	if err := CheckPassword(h, pw); err != nil {
		t.Fatalf("CheckPassword failed: %v", err)
	}
	if err := CheckPassword(h, "wrong"); err == nil {
		t.Fatalf("expected mismatch error for wrong password")
	}
	if _, err := HashPassword(""); err == nil {
		t.Fatalf("expected error when hashing empty password")
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-123"}
	a := New(cfg)

	token, err := a.GenerateToken("42", "admin", time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	claims, err := a.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if claims.UserID != "42" || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	// Invalid signature: sign with different secret
	tkn := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "1", Role: "x"})
	bad, _ := tkn.SignedString([]byte("wrong-secret"))
	if _, err := a.ParseToken(bad); err == nil {
		t.Fatalf("expected error parsing token with wrong signature")
	}

	// Expired token: create token with past expiry
	past := time.Now().Add(-time.Hour)
	expClaims := Claims{UserID: "5", Role: "u", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(past), IssuedAt: jwt.NewNumericDate(past.Add(-time.Hour))}}
	tkn2 := jwt.NewWithClaims(jwt.SigningMethodHS256, expClaims)
	signed, _ := tkn2.SignedString([]byte(cfg.JWTSecret))
	if _, err := a.ParseToken(signed); err == nil {
		t.Fatalf("expected error parsing expired token")
	}
}
