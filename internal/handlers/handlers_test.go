package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mayvqt/Sentinel/internal/auth"
	"github.com/mayvqt/Sentinel/internal/config"
	"github.com/mayvqt/Sentinel/internal/models"
	"github.com/mayvqt/Sentinel/internal/store"
)

func setupTestHandlers() (*Handlers, store.Store) {
	s := store.NewMemStore()
	cfg := &config.Config{JWTSecret: "test-secret-123"}
	a := auth.New(cfg)
	h := New(s, a)
	return h, s
}

func TestRegisterLoginHealth(t *testing.T) {
	s := store.NewMemStore()
	cfg := &config.Config{JWTSecret: "test-secret"}
	a := auth.New(cfg)
	h := New(s, a)

	// Register a user - updated with email and stronger password
	regPayload := map[string]string{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "SecurePass123!",
	}
	b, _ := json.Marshal(regPayload)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	// Login
	loginPayload := map[string]string{"username": "alice", "password": "SecurePass123!"}
	lb, _ := json.Marshal(loginPayload)
	lr := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(lb))
	lr.Header.Set("Content-Type", "application/json")
	lw := httptest.NewRecorder()
	h.Login(lw, lr)
	lres := lw.Result()
	if lres.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d", lres.StatusCode)
	}
	body, _ := io.ReadAll(lres.Body)
	var loginResponse struct {
		AccessToken  string      `json:"access_token"`
		RefreshToken string      `json:"refresh_token"`
		TokenType    string      `json:"token_type"`
		ExpiresIn    int         `json:"expires_in"`
		User         models.User `json:"user"`
	}
	_ = json.Unmarshal(body, &loginResponse)
	if loginResponse.AccessToken == "" {
		t.Fatalf("expected access_token in login response")
	}
	if loginResponse.RefreshToken == "" {
		t.Fatalf("expected refresh_token in login response")
	}
	if loginResponse.TokenType != "Bearer" {
		t.Fatalf("expected token_type to be Bearer, got %s", loginResponse.TokenType)
	}

	// Test /me endpoint
	meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)

	// Add user context (normally done by auth middleware)
	claims := &auth.Claims{
		UserID: "1",
		Role:   "user",
	}
	ctx := context.WithValue(meReq.Context(), "user", claims)
	meReq = meReq.WithContext(ctx)

	meW := httptest.NewRecorder()
	h.Me(meW, meReq)
	if meW.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /me, got %d", meW.Result().StatusCode)
	}

	// Health
	hw := httptest.NewRecorder()
	hr := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.Health(hw, hr)
	if hw.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from health, got %d", hw.Result().StatusCode)
	}

	_ = s.Close()
}

func TestRegisterValidation(t *testing.T) {
	h, _ := setupTestHandlers()

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
	}{
		{
			name: "valid registration",
			payload: map[string]string{
				"username": "testuser",
				"email":    "test@example.com",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "weak password",
			payload: map[string]string{
				"username": "testuser2",
				"email":    "test2@example.com",
				"password": "weak",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid email",
			payload: map[string]string{
				"username": "testuser3",
				"email":    "invalid-email",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "reserved username",
			payload: map[string]string{
				"username": "admin",
				"email":    "admin@example.com",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "short username",
			payload: map[string]string{
				"username": "ab",
				"email":    "test@example.com",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Register(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Register() status = %v, want %v, body: %s",
					w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

func TestLoginEdgeCases(t *testing.T) {
	h, s := setupTestHandlers()

	// Create a test user
	hashedPassword, _ := auth.HashPassword("SecurePass123!")
	user := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  hashedPassword,
		Role:      "user",
		CreatedAt: time.Now(),
	}
	_, err := s.CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
	}{
		{
			name: "valid login",
			payload: map[string]string{
				"username": "testuser",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "wrong password",
			payload: map[string]string{
				"username": "testuser",
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "nonexistent user",
			payload: map[string]string{
				"username": "nonexistent",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "empty username",
			payload: map[string]string{
				"username": "",
				"password": "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty password",
			payload: map[string]string{
				"username": "testuser",
				"password": "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Login() status = %v, want %v, body: %s",
					w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

func TestMeEndpoint(t *testing.T) {
	h, s := setupTestHandlers()

	// Create a test user
	user := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "hashedpassword",
		Role:      "user",
		CreatedAt: time.Now(),
	}
	userID, err := s.CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name           string
		userClaims     *auth.Claims
		expectedStatus int
	}{
		{
			name: "valid user context",
			userClaims: &auth.Claims{
				UserID: "1",
				Role:   "user",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no user context",
			userClaims:     nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid user ID format",
			userClaims: &auth.Claims{
				UserID: "invalid",
				Role:   "user",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/me", nil)

			if tt.userClaims != nil {
				// For the valid case, use the actual user ID
				if tt.name == "valid user context" {
					tt.userClaims.UserID = "1" // We know the first user gets ID 1
				}
				ctx := context.WithValue(req.Context(), "user", tt.userClaims)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			h.Me(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Me() status = %v, want %v, userID: %v, body: %s",
					w.Code, tt.expectedStatus, userID, w.Body.String())
			}
		})
	}
}
