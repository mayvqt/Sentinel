// Package middleware provides HTTP middleware utilities.
package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// RateLimiter is a token-bucket limiter optimized for concurrency.
type RateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*visitor
	rate     time.Duration // Time between requests
	capacity int           // Maximum burst capacity
	stopChan chan struct{} // Channel to stop cleanup goroutine
	stopped  int32         // Atomic flag to indicate if stopped
}

type visitor struct {
	mu       sync.Mutex
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
		stopChan: make(chan struct{}),
		stopped:  0,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Stop gracefully stops the rate limiter cleanup goroutine.
func (rl *RateLimiter) Stop() {
	if atomic.CompareAndSwapInt32(&rl.stopped, 0, 1) {
		close(rl.stopChan)
	}
}

// Allow checks if a request should be allowed based on the client IP.
// Uses fine-grained locking for better concurrency.
func (rl *RateLimiter) Allow(ip string) bool {
	now := time.Now()

	// Try to get existing visitor with read lock first
	rl.mu.RLock()
	v, exists := rl.visitors[ip]
	rl.mu.RUnlock()

	if !exists {
		// Create new visitor with write lock
		rl.mu.Lock()
		// Double-check in case another goroutine created it
		v, exists = rl.visitors[ip]
		if !exists {
			v = &visitor{
				lastSeen: now,
				tokens:   rl.capacity - 1, // Use one token
			}
			rl.visitors[ip] = v
			rl.mu.Unlock()
			return true
		}
		rl.mu.Unlock()
	}

	// Lock the specific visitor for thread-safe token updates
	v.mu.Lock()
	defer v.mu.Unlock()

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
// Runs periodically until Stop() is called.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanupVisitors()
		case <-rl.stopChan:
			return
		}
	}
}

// cleanupVisitors removes stale visitor entries.
func (rl *RateLimiter) cleanupVisitors() {
	cutoff := time.Now().Add(-10 * time.Minute)
	toDelete := make([]string, 0)

	// Collect IPs to delete with read lock
	rl.mu.RLock()
	for ip, v := range rl.visitors {
		v.mu.Lock()
		if v.lastSeen.Before(cutoff) {
			toDelete = append(toDelete, ip)
		}
		v.mu.Unlock()
	}
	rl.mu.RUnlock()

	// Delete with write lock if needed
	if len(toDelete) > 0 {
		rl.mu.Lock()
		for _, ip := range toDelete {
			delete(rl.visitors, ip)
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
