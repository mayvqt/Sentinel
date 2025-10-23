// Package handlers provides HTTP handlers and route wiring.
//
// Handlers should be thin: validate input, call services/store, and write
// responses. Keep business logic out of handlers to make testing easier.
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mayvqt/Sentinel/internal/auth"
	"github.com/mayvqt/Sentinel/internal/models"
	"github.com/mayvqt/Sentinel/internal/store"
	"github.com/mayvqt/Sentinel/internal/validation"
)

// Handlers holds dependencies for HTTP endpoints.
type Handlers struct {
	Store store.Store
	Auth  *auth.Auth
}

// New constructs handlers with dependencies injected.
func New(s store.Store, a *auth.Auth) *Handlers {
	return &Handlers{Store: s, Auth: a}
}

// registerRequest is the expected payload for POST /register.
type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginRequest is the expected payload for POST /login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register creates a new user. This stub performs minimal validation,
// hashes the password, stores the user and returns 201 with a small JSON
// body.
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password required", http.StatusBadRequest)
		return
	}

	// Hash password
	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	u := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashed,
		Role:      "user",
		CreatedAt: time.Now().UTC(),
	}

	id, err := h.Store.CreateUser(r.Context(), u)
	if err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": id})
}

// Login validates credentials and returns a JWT on success.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	u, err := h.Store.GetUserByUsername(r.Context(), req.Username)
	if err != nil || u == nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := auth.CheckPassword(u.Password, req.Password); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	// Generate token with a 24h TTL for now.
	tok, err := h.Auth.GenerateToken(strconv.FormatInt(u.ID, 10), u.Role, 24*time.Hour)
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"token": tok})
}

// Health returns a simple alive response.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
