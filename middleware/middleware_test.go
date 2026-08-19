package middleware_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"teachar.in/middleware"
	"teachar.in/repository"
	"teachar.in/services"
)

func setupTestMiddleware(t *testing.T) *middleware.Manager {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := repository.NewSQLiteRepository(dbPath, tempDir)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	authService := services.NewAuthService(repo)
	securityService := services.NewSecurityService(repo)
	return middleware.NewManager(logger, authService, securityService)
}

func TestMiddlewareChainAndSecurity(t *testing.T) {
	mw := setupTestMiddleware(t)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	chained := mw.Chain(mw.Recovery, mw.Logging, mw.Security, mw.AuthenticateSession)(finalHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	chained.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", res.StatusCode)
	}

	if res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff header")
	}
	if res.Header.Get("X-Frame-Options") != "deny" {
		t.Errorf("expected X-Frame-Options: deny header")
	}
}

func TestMiddlewareRecovery(t *testing.T) {
	mw := setupTestMiddleware(t)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	chained := mw.Recovery(panicHandler)

	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()

	chained.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 Internal Server Error after panic, got %d", res.StatusCode)
	}
}
