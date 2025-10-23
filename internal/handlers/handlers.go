// Package handlers contains HTTP handlers. Keep handlers thin: validate
// input, call services/store, and write responses.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mayvqt/Sentinel/internal/auth"
	"github.com/mayvqt/Sentinel/internal/logger"
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

// refreshRequest is the expected payload for POST /refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Register creates a new user with comprehensive validation and security checks.
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	log := logger.WithFields(map[string]interface{}{
		"handler":  "register",
		"method":   r.Method,
		"username": "",
		"email":    "",
	})

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid JSON payload in registration request", map[string]interface{}{
			"handler": "register",
			"error":   err.Error(),
		})
		writeErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Sanitize inputs
	req.Username = validation.SanitizeInput(req.Username)
	req.Email = validation.SanitizeInput(req.Email)
	req.Password = validation.SanitizeInput(req.Password)

	log = logger.WithFields(map[string]interface{}{
		"handler":  "register",
		"username": req.Username,
		"email":    req.Email,
	})

	// Validate the registration request
	if err := validation.ValidateRegisterRequest(req.Username, req.Email, req.Password); err != nil {
		log.Warn("Registration validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if user already exists
	existingUser, err := h.Store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		log.Error("Database error while checking existing user", map[string]interface{}{
			"error": err.Error(),
		})
		writeErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existingUser != nil {
		log.Warn("Registration attempt with existing username")
		writeErrorResponse(w, "Username already exists", http.StatusConflict)
		return
	}

	// Hash password with strong settings
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Error("Password hashing failed", map[string]interface{}{
			"error": err.Error(),
		})
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
			log.Warn("User creation failed due to duplicate", map[string]interface{}{
				"error": err.Error(),
			})
			writeErrorResponse(w, err.Error(), http.StatusConflict)
			return
		}
		log.Error("User creation failed", map[string]interface{}{
			"error": err.Error(),
		})
		writeErrorResponse(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	log.Info("User successfully registered", map[string]interface{}{
		"user_id": userID,
	})

	// Return success response with user ID (no sensitive data)
	response := map[string]interface{}{
		"id":      userID,
		"message": "User created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
} // Login validates credentials and returns a JWT on success with rate limiting considerations.
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

	// Generate access token (1 hour) and refresh token (7 days)
	accessToken, err := h.Auth.GenerateTokenWithType(
		strconv.FormatInt(user.ID, 10),
		user.Role,
		"access",
		1*time.Hour,
	)
	if err != nil {
		writeErrorResponse(w, "Failed to create authentication token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := h.Auth.GenerateTokenWithType(
		strconv.FormatInt(user.ID, 10),
		user.Role,
		"refresh",
		7*24*time.Hour,
	)
	if err != nil {
		writeErrorResponse(w, "Failed to create refresh token", http.StatusInternalServerError)
		return
	}

	// Return tokens and basic user info (no sensitive data)
	response := map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    3600, // 1 hour in seconds
		"user":          user.PublicUser(),
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

// RefreshToken exchanges a valid refresh token for new access and refresh tokens.
// This implements token rotation for enhanced security.
func (h *Handlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate refresh token
	claims, err := h.Auth.ParseToken(req.RefreshToken)
	if err != nil {
		writeErrorResponse(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	// Verify token type
	if claims.TokenType != "refresh" {
		writeErrorResponse(w, "Token is not a refresh token", http.StatusBadRequest)
		return
	}

	// Parse user ID
	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		writeErrorResponse(w, "Invalid user ID in token", http.StatusBadRequest)
		return
	}

	// Verify user still exists
	user, err := h.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if user == nil {
		writeErrorResponse(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Generate new access token and refresh token (token rotation)
	newAccessToken, err := h.Auth.GenerateTokenWithType(
		claims.UserID,
		claims.Role,
		"access",
		1*time.Hour,
	)
	if err != nil {
		writeErrorResponse(w, "Failed to create access token", http.StatusInternalServerError)
		return
	}

	newRefreshToken, err := h.Auth.GenerateTokenWithType(
		claims.UserID,
		claims.Role,
		"refresh",
		7*24*time.Hour,
	)
	if err != nil {
		writeErrorResponse(w, "Failed to create refresh token", http.StatusInternalServerError)
		return
	}

	// Return new tokens
	response := map[string]interface{}{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    3600, // 1 hour in seconds
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
