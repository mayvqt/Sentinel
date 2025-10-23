package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mayvqt/Sentinel/internal/auth"
	"github.com/mayvqt/Sentinel/internal/middleware"
	"github.com/mayvqt/Sentinel/internal/config"
	"github.com/mayvqt/Sentinel/internal/store"
)

func TestRegisterLoginHealth(t *testing.T) {
	s := store.NewMemStore()
	cfg := &config.Config{JWTSecret: "test-secret"}
	a := auth.New(cfg)
	h := New(s, a)

	// Register a user
	regPayload := map[string]string{"username": "alice", "password": "pw123"}
	b, _ := json.Marshal(regPayload)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(b))
	w := httptest.NewRecorder()
	h.Register(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	// Login
	loginPayload := map[string]string{"username": "alice", "password": "pw123"}
	lb, _ := json.Marshal(loginPayload)
	lr := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(lb))
	lw := httptest.NewRecorder()
	h.Login(lw, lr)
	lres := lw.Result()
	if lres.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d", lres.StatusCode)
	}
	body, _ := io.ReadAll(lres.Body)
	var obj map[string]string
	_ = json.Unmarshal(body, &obj)
	tok, ok := obj["token"]
	if !ok || tok == "" {
		t.Fatalf("expected token in login response")
	}

	// Protected /me using middleware
	mux := http.NewServeMux()
	mux.Handle("/me", middleware.WithAuth(a)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("me"))
	})))

	r := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	mw := httptest.NewRecorder()
	mux.ServeHTTP(mw, r)
	if mw.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /me, got %d", mw.Result().StatusCode)
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
