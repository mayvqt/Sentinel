// Package handlers provides HTTP handlers and route wiring.
//
// Handlers should be thin: validate input, call services/store, and write
// responses. Keep business logic out of handlers to make testing easier.
package handlers

import (
	"encoding/json"
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

// ErrorResponse represents a structured error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// writeErrorResponse writes a structured error response.
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
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

// Register creates a new user with comprehensive validation and security checks.
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Sanitize inputs
	req.Username = validation.SanitizeInput(req.Username)
	req.Email = validation.SanitizeInput(req.Email)
	req.Password = validation.SanitizeInput(req.Password)

	// Validate the registration request
	if err := validation.ValidateRegisterRequest(req.Username, req.Email, req.Password); err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if user already exists
	existingUser, err := h.Store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		writeErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existingUser != nil {
		writeErrorResponse(w, "Username already exists", http.StatusConflict)
		return
	}

	// Hash password with strong settings
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErrorResponse(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Create user
	user := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword,
		Role:      "user",
		CreatedAt: time.Now().UTC(),
	}

	userID, err := h.Store.CreateUser(r.Context(), user)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeErrorResponse(w, err.Error(), http.StatusConflict)
			return
		}
		writeErrorResponse(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Return success response with user ID (no sensitive data)
	response := map[string]interface{}{
		"id":      userID,
		"message": "User created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Login validates credentials and returns a JWT on success with rate limiting considerations.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Sanitize inputs
	req.Username = validation.SanitizeInput(req.Username)
	req.Password = validation.SanitizeInput(req.Password)

	// Basic validation
	if req.Username == "" || req.Password == "" {
		writeErrorResponse(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Get user from store
	user, err := h.Store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		writeErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if user exists and verify password
	if user == nil || auth.CheckPassword(user.Password, req.Password) != nil {
		// Use the same error message for both cases to prevent username enumeration
		writeErrorResponse(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token with 24-hour expiration
	token, err := h.Auth.GenerateToken(
		strconv.FormatInt(user.ID, 10),
		user.Role,
		24*time.Hour,
	)
	if err != nil {
		writeErrorResponse(w, "Failed to create authentication token", http.StatusInternalServerError)
		return
	}

	// Return token and basic user info (no sensitive data)
	response := map[string]interface{}{
		"token": token,
		"user":  user.PublicUser(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Health returns system health status with basic diagnostics.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	if err := h.Store.Ping(r.Context()); err != nil {
		writeErrorResponse(w, "Database unavailable", http.StatusServiceUnavailable)
		return
	}

	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "0.1.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Me returns the current authenticated user's profile.
func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	// Extract user claims from context (set by auth middleware)
	claims, ok := r.Context().Value("user").(*auth.Claims)
	if !ok {
		writeErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Parse user ID from claims
	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		writeErrorResponse(w, "Invalid user ID in token", http.StatusBadRequest)
		return
	}

	// Get user from store
	user, err := h.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if user == nil {
		writeErrorResponse(w, "User not found", http.StatusNotFound)
		return
	}

	// Return user profile (excluding sensitive data)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user.PublicUser())
}
