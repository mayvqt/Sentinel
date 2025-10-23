// Package validation provides input validation utilities with security best practices.
package validation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	// Email validation regex - RFC 5322 compliant
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// Username validation regex - alphanumeric, underscore, hyphen, 3-32 chars
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)
)

// ValidationError represents a validation error with a user-friendly message.
type ValidationError struct {
	Field   string
	Message string
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

// ValidationErrors represents multiple validation errors.
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	if len(ve) == 1 {
		return ve[0].Error()
	}

	var messages []string
	for _, err := range ve {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// ValidateEmail validates email format and length.
func ValidateEmail(email string) error {
	if email == "" {
		return ValidationError{Field: "email", Message: "email is required"}
	}

	if len(email) > 254 {
		return ValidationError{Field: "email", Message: "email must be less than 255 characters"}
	}

	if !emailRegex.MatchString(email) {
		return ValidationError{Field: "email", Message: "email format is invalid"}
	}

	return nil
}

// ValidateUsername validates username format, length, and content.
func ValidateUsername(username string) error {
	if username == "" {
		return ValidationError{Field: "username", Message: "username is required"}
	}

	if len(username) < 3 {
		return ValidationError{Field: "username", Message: "username must be at least 3 characters"}
	}

	if len(username) > 32 {
		return ValidationError{Field: "username", Message: "username must be less than 33 characters"}
	}

	if !usernameRegex.MatchString(username) {
		return ValidationError{Field: "username", Message: "username can only contain letters, numbers, underscores, and hyphens"}
	}

	// Prevent reserved usernames
	reserved := []string{"admin", "root", "user", "api", "www", "mail", "system", "support", "null", "undefined"}
	lowerUsername := strings.ToLower(username)
	for _, r := range reserved {
		if lowerUsername == r {
			return ValidationError{Field: "username", Message: "username is reserved"}
		}
	}

	return nil
}

// ValidatePassword validates password strength using comprehensive criteria.
func ValidatePassword(password string) error {
	if password == "" {
		return ValidationError{Field: "password", Message: "password is required"}
	}

	if len(password) < 8 {
		return ValidationError{Field: "password", Message: "password must be at least 8 characters"}
	}

	if len(password) > 128 {
		return ValidationError{Field: "password", Message: "password must be less than 129 characters"}
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	var missing []string
	if !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if !hasNumber {
		missing = append(missing, "number")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		return ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("password must contain at least one: %s", strings.Join(missing, ", ")),
		}
	}

	// Check for common weak patterns
	if isCommonPassword(password) {
		return ValidationError{Field: "password", Message: "password is too common"}
	}

	return nil
}

// ValidateRole validates user role.
func ValidateRole(role string) error {
	if role == "" {
		return ValidationError{Field: "role", Message: "role is required"}
	}

	validRoles := []string{"user", "admin", "moderator"}
	for _, validRole := range validRoles {
		if role == validRole {
			return nil
		}
	}

	return ValidationError{Field: "role", Message: "invalid role"}
}

// isCommonPassword checks against a list of common weak passwords.
func isCommonPassword(password string) bool {
	commonPasswords := []string{
		"password", "123456", "123456789", "12345678", "12345", "1234567",
		"admin", "letmein", "welcome", "monkey", "1234567890", "qwerty",
		"abc123", "Password1", "password123", "admin123", "root", "toor",
	}

	lowerPassword := strings.ToLower(password)
	for _, common := range commonPasswords {
		if lowerPassword == strings.ToLower(common) {
			return true
		}
	}

	return false
}

// ValidateRegisterRequest validates a complete registration request.
func ValidateRegisterRequest(username, email, password string) error {
	var errs ValidationErrors

	if err := ValidateUsername(username); err != nil {
		if ve, ok := err.(ValidationError); ok {
			errs = append(errs, ve)
		} else {
			errs = append(errs, ValidationError{Field: "username", Message: err.Error()})
		}
	}

	if err := ValidateEmail(email); err != nil {
		if ve, ok := err.(ValidationError); ok {
			errs = append(errs, ve)
		} else {
			errs = append(errs, ValidationError{Field: "email", Message: err.Error()})
		}
	}

	if err := ValidatePassword(password); err != nil {
		if ve, ok := err.(ValidationError); ok {
			errs = append(errs, ve)
		} else {
			errs = append(errs, ValidationError{Field: "password", Message: err.Error()})
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// SanitizeInput removes potentially dangerous characters from user input.
func SanitizeInput(input string) string {
	// Remove null bytes and control characters
	cleaned := strings.Map(func(r rune) rune {
		if r == 0 || (r < 32 && r != '\t' && r != '\n' && r != '\r') {
			return -1
		}
		return r
	}, input)

	// Trim whitespace
	return strings.TrimSpace(cleaned)
}
