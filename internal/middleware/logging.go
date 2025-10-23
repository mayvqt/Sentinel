package middleware

import (
	"net/http"
	"time"

	"github.com/mayvqt/Sentinel/internal/logger"
)

// responseWriter records status and response size for logging.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// WithLogging returns middleware that logs HTTP requests.
func WithLogging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the response writer
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     0,
			}

			// Get client IP
			clientIP := getClientIP(r)

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log request details
			duration := time.Since(start)

			fields := map[string]interface{}{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status_code": wrapped.statusCode,
				"duration_ms": duration.Milliseconds(),
				"client_ip":   clientIP,
				"user_agent":  r.UserAgent(),
				"bytes":       wrapped.written,
			}

			// Add request ID if available
			if requestID := GetRequestID(r.Context()); requestID != "" {
				fields["request_id"] = requestID
			}

			// Add query parameters if present
			if r.URL.RawQuery != "" {
				fields["query"] = r.URL.RawQuery
			}

			// Log level based on status code
			message := "HTTP request processed"
			if wrapped.statusCode >= 500 {
				logger.Error(message, fields)
			} else if wrapped.statusCode >= 400 {
				logger.Warn(message, fields)
			} else {
				logger.Info(message, fields)
			}
		})
	}
}
