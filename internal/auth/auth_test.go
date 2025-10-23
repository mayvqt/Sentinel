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

func TestAuthWithNoSecret(t *testing.T) {
	a := New(nil) // No config

	_, err := a.GenerateToken("123", "user", time.Hour)
	if err != ErrNoSecret {
		t.Errorf("GenerateToken() with no secret should return ErrNoSecret, got %v", err)
	}

	_, err = a.ParseToken("some.token.here")
	if err != ErrNoSecret {
		t.Errorf("ParseToken() with no secret should return ErrNoSecret, got %v", err)
	}
}

func TestGenerateTokenValidation(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-123"}
	a := New(cfg)

	tests := []struct {
		name    string
		userID  string
		role    string
		ttl     time.Duration
		wantErr bool
	}{
		{"valid token", "123", "user", time.Hour, false},
		{"zero ttl", "123", "user", 0, true},
		{"negative ttl", "123", "user", -time.Hour, true},
		{"empty userID", "", "user", time.Hour, false}, // Currently allowed
		{"empty role", "123", "", time.Hour, false},    // Currently allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := a.GenerateToken(tt.userID, tt.role, tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseTokenEdgeCases(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-123"}
	a := New(cfg)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"empty token", "", true},
		{"malformed token", "not.a.jwt", true},
		{"invalid base64", "invalid.base64.token", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := a.ParseToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "testpassword123"
	for i := 0; i < b.N; i++ {
		HashPassword(password)
	}
}

func BenchmarkCheckPassword(b *testing.B) {
	password := "testpassword123"
	hash, _ := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckPassword(hash, password)
	}
}

func BenchmarkGenerateToken(b *testing.B) {
	cfg := &config.Config{JWTSecret: "test-secret-123"}
	a := New(cfg)

	for i := 0; i < b.N; i++ {
		a.GenerateToken("123", "user", time.Hour)
	}
}

func BenchmarkParseToken(b *testing.B) {
	cfg := &config.Config{JWTSecret: "test-secret-123"}
	a := New(cfg)
	token, _ := a.GenerateToken("123", "user", time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.ParseToken(token)
	}
}
