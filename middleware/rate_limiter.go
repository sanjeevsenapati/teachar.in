package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type clientStats struct {
	count     int
	resetTime time.Time
}

// RateLimiter enforces request rate limits per IP address or API key.
type RateLimiter struct {
	mu            sync.Mutex
	clients       map[string]*clientStats
	limit         int
	windowSeconds time.Duration
}

func NewRateLimiter(limit int, windowSeconds time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		clients:       make(map[string]*clientStats),
		limit:         limit,
		windowSeconds: windowSeconds,
	}

	// Periodic cleanup of stale client IPs
	go func() {
		for {
			time.Sleep(windowSeconds * 2)
			limiter.cleanup()
		}
	}()

	return limiter
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, stats := range rl.clients {
		if now.After(stats.resetTime) {
			delete(rl.clients, ip)
		}
	}
}

// LimitMiddleware wraps an http.Handler with rate limiting enforcement.
func (rl *RateLimiter) LimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		if r.URL.Path == "/api/staff/create" {
			next.ServeHTTP(w, r)
			return
		}

		rl.mu.Lock()
		now := time.Now()
		stats, exists := rl.clients[ip]
		if !exists || now.After(stats.resetTime) {
			rl.clients[ip] = &clientStats{
				count:     1,
				resetTime: now.Add(rl.windowSeconds),
			}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		stats.count++
		if stats.count > rl.limit {
			rl.mu.Unlock()
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded. Please wait a minute before retrying.", http.StatusTooManyRequests)
			return
		}
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
