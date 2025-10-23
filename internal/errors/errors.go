// Package errors provides custom error types for enterprise-grade error handling.
package errors

import (
	"errors"
	"fmt"
)

// ErrorCode represents a specific error type for better error handling.
type ErrorCode string

const (
	// Authentication errors
	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	ErrCodeTokenInvalid       ErrorCode = "TOKEN_INVALID"
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"

	// Validation errors
	ErrCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrCodeInvalidInput   ErrorCode = "INVALID_INPUT"
	ErrCodeMissingField   ErrorCode = "MISSING_FIELD"
	ErrCodeDuplicateEntry ErrorCode = "DUPLICATE_ENTRY"

	// Database errors
	ErrCodeDatabase   ErrorCode = "DATABASE_ERROR"
	ErrCodeNotFound   ErrorCode = "NOT_FOUND"
	ErrCodeConflict   ErrorCode = "CONFLICT"
	ErrCodeTimeout    ErrorCode = "TIMEOUT"
	ErrCodeConnection ErrorCode = "CONNECTION_ERROR"

	// Rate limiting errors
	ErrCodeRateLimit ErrorCode = "RATE_LIMIT_EXCEEDED"

	// Server errors
	ErrCodeInternal      ErrorCode = "INTERNAL_ERROR"
	ErrCodeUnavailable   ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeBadRequest    ErrorCode = "BAD_REQUEST"
	ErrCodeNotImplemented ErrorCode = "NOT_IMPLEMENTED"
)

// AppError represents an application-specific error with additional context.
type AppError struct {
	Code    ErrorCode              // Machine-readable error code
	Message string                 // Human-readable error message
	Err     error                  // Original error (if any)
	Fields  map[string]interface{} // Additional context fields
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithField adds a context field to the error.
func (e *AppError) WithField(key string, value interface{}) *AppError {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[key] = value
	return e
}

// New creates a new AppError.
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Fields:  make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with an AppError.
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Fields:  make(map[string]interface{}),
	}
}

// IsAppError checks if an error is an AppError.
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetCode extracts the error code from an error if it's an AppError.
func GetCode(err error) ErrorCode {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ErrCodeInternal
}

// Common error constructors for convenience

// ErrInvalidCredentials creates an invalid credentials error.
func ErrInvalidCredentials() *AppError {
	return New(ErrCodeInvalidCredentials, "Invalid username or password")
}

// ErrTokenExpired creates a token expired error.
func ErrTokenExpired() *AppError {
	return New(ErrCodeTokenExpired, "Token has expired")
}

// ErrTokenInvalid creates an invalid token error.
func ErrTokenInvalid() *AppError {
	return New(ErrCodeTokenInvalid, "Token is invalid")
}

// ErrUnauthorized creates an unauthorized error.
func ErrUnauthorized(message string) *AppError {
	if message == "" {
		message = "Unauthorized access"
	}
	return New(ErrCodeUnauthorized, message)
}

// ErrValidation creates a validation error.
func ErrValidation(message string) *AppError {
	return New(ErrCodeValidation, message)
}

// ErrNotFound creates a not found error.
func ErrNotFound(resource string) *AppError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s not found", resource))
}

// ErrDuplicate creates a duplicate entry error.
func ErrDuplicate(resource string) *AppError {
	return New(ErrCodeDuplicateEntry, fmt.Sprintf("%s already exists", resource))
}

// ErrDatabase creates a database error.
func ErrDatabase(err error, message string) *AppError {
	return Wrap(err, ErrCodeDatabase, message)
}

// ErrRateLimit creates a rate limit error.
func ErrRateLimit() *AppError {
	return New(ErrCodeRateLimit, "Rate limit exceeded, please try again later")
}

// ErrInternal creates an internal server error.
func ErrInternal(err error, message string) *AppError {
	if message == "" {
		message = "Internal server error"
	}
	return Wrap(err, ErrCodeInternal, message)
}
