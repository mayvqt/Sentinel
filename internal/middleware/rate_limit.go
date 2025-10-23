// Package middleware provides HTTP middleware for security and rate limiting.
package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter.
type RateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*visitor
	rate     time.Duration // Time between requests
	capacity int           // Maximum burst capacity
}

type visitor struct {
	lastSeen time.Time
	tokens   int
}

// NewRateLimiter creates a new rate limiter.
// rate: minimum time between requests (e.g., time.Second for 1 req/sec)
// capacity: maximum burst requests allowed
func NewRateLimiter(rate time.Duration, capacity int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		capacity: capacity,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request should be allowed based on the client IP.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]

	if !exists {
		rl.visitors[ip] = &visitor{
			lastSeen: now,
			tokens:   rl.capacity - 1, // Use one token
		}
		return true
	}

	// Add tokens based on time elapsed
	elapsed := now.Sub(v.lastSeen)
	tokensToAdd := int(elapsed / rl.rate)

	if tokensToAdd > 0 {
		v.tokens += tokensToAdd
		if v.tokens > rl.capacity {
			v.tokens = rl.capacity
		}
		v.lastSeen = now
	}

	// Check if we can consume a token
	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

// cleanup removes old visitor entries to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)

		for ip, v := range rl.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// WithRateLimit returns middleware that enforces rate limiting.
func WithRateLimit(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP
			ip := getClientIP(r)

			if !rl.Allow(ip) {
				writeRateLimitError(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for requests behind proxy)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP in the list
		if ip := net.ParseIP(forwarded); ip != nil {
			return ip.String()
		}
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		if ip := net.ParseIP(realIP); ip != nil {
			return ip.String()
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// writeRateLimitError writes a rate limit exceeded error response.
func writeRateLimitError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60") // Suggest retry after 60 seconds
	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]string{
		"error":   "Too Many Requests",
		"message": "Rate limit exceeded. Please try again later.",
	}

	json.NewEncoder(w).Encode(response)
}
