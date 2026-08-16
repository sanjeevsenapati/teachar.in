package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"teachar.in/models"
	"teachar.in/services"
)

type contextKey string

const (
	UserContextKey   contextKey = "authenticated_user"
	APIKeyContextKey contextKey = "authenticated_apikey"
)

// Manager holds dependencies for middleware, like a logger, auth service, and security service.
type Manager struct {
	logger          *slog.Logger
	authService     *services.AuthService
	securityService *services.SecurityService
}

// NewManager creates a new middleware manager.
func NewManager(logger *slog.Logger, authService *services.AuthService, securityService ...*services.SecurityService) *Manager {
	mgr := &Manager{logger: logger, authService: authService}
	if len(securityService) > 0 {
		mgr.securityService = securityService[0]
	}
	return mgr
}

// Middleware is a function that takes a http.Handler and returns a http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies a series of middleware to a http.Handler.
func (m *Manager) Chain(middlewares ...Middleware) func(http.Handler) http.Handler {
	return func(finalHandler http.Handler) http.Handler {
		last := finalHandler
		for i := len(middlewares) - 1; i >= 0; i-- {
			last = middlewares[i](last)
		}
		return last
	}
}

// AuthenticateSession attaches user data to request context if session cookie is valid.
func (m *Manager) AuthenticateSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err == nil && cookie.Value != "" {
			_, user, err := m.authService.ValidateSession(r.Context(), cookie.Value)
			if err == nil && user != nil {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAuth blocks unauthenticated users and redirects them to /login.
func (m *Manager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r)
		if user == nil {
			http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAPIKey checks X-API-Key or Authorization Bearer header for valid API key authentication.
func (m *Manager) RequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.securityService == nil {
			next.ServeHTTP(w, r)
			return
		}

		apiKeyHeader := r.Header.Get("X-API-Key")
		if apiKeyHeader == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKeyHeader = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKeyHeader == "" {
			http.Error(w, `{"error":"missing API key header 'X-API-Key' or 'Authorization: Bearer <key>'"}`, http.StatusUnauthorized)
			return
		}

		key, err := m.securityService.ValidateAPIKey(r.Context(), apiKeyHeader)
		if err != nil || key == nil {
			http.Error(w, `{"error":"invalid, expired, or revoked API key"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), APIKeyContextKey, key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin blocks non-admin users (allows "admin" and "superadmin").
func (m *Manager) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r)
		if user == nil || (user.Role != "admin" && user.Role != "superadmin") {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireSuperadmin strictly requires the "superadmin" role.
func (m *Manager) RequireSuperadmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r)
		if user == nil || user.Role != "superadmin" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireStaffOrAdmin allows staff, admin, and superadmin users.
func (m *Manager) RequireStaffOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r)
		if user == nil || (user.Role != "staff" && user.Role != "admin" && user.Role != "superadmin") {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext extracts the authenticated User from request context.
func GetUserFromContext(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// Logging logs information about each incoming request.
func (m *Manager) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		m.logger.Info("request handled", "method", r.Method, "url", r.URL.Path, "duration", time.Since(start))
	})
}

// Recovery recovers from panics and returns a 500 Internal Server Error.
func (m *Manager) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				m.logger.Error(fmt.Sprintf("%s", err), "method", r.Method, "uri", r.URL.RequestURI())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Security adds enterprise-grade security headers to every HTTP response.
func (m *Manager) Security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}
