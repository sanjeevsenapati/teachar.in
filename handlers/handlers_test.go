package handlers_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"teachar.in/config"
	"teachar.in/handlers"
	"teachar.in/repository"
	"teachar.in/services"
)

func setupTestApp(t *testing.T) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg, _ := config.New()
	
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_db.json")
	repo, err := repository.NewJSONRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}

	couponService := services.NewCouponService(repo)
	menuService := services.NewMenuService(repo)
	authService := services.NewAuthService(repo)
	orderService := services.NewOrderService(repo, couponService)
	auditService := services.NewAuditService(repo)
	reportService := services.NewReportService(repo, repo)

	app := &handlers.Application{
		Logger:        logger,
		Config:        cfg,
		MenuService:   menuService,
		AuthService:   authService,
		OrderService:  orderService,
		AuditService:  auditService,
		ReportService: reportService,
		CouponService: couponService,
	}

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	return mux
}

func TestHealthCheckHandler(t *testing.T) {
	handler := setupTestApp(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var data map[string]string
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if data["status"] != "UP" {
		t.Errorf("expected status UP, got %s", data["status"])
	}
}

func TestAPIStatusHandler(t *testing.T) {
	handler := setupTestApp(t)

	req := httptest.NewRequest("GET", "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}
}

func TestAPIGetMenuHandler(t *testing.T) {
	handler := setupTestApp(t)

	req := httptest.NewRequest("GET", "/api/menu", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}
}

func TestAPIGetMenuItemHandler(t *testing.T) {
	handler := setupTestApp(t)

	t.Run("Valid Item", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/menu/1", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		if res.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", res.StatusCode)
		}
	})

	t.Run("Invalid Item", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/menu/999", nil)
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", res.StatusCode)
		}
	})
}

func TestPageHandlers(t *testing.T) {
	handler := setupTestApp(t)

	pages := []string{"/", "/menu", "/about", "/login", "/register"}
	for _, page := range pages {
		t.Run(page, func(t *testing.T) {
			req := httptest.NewRequest("GET", page, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			res := rec.Result()
			if res.StatusCode != http.StatusOK {
				t.Errorf("page %s: expected status 200, got %d", page, res.StatusCode)
			}
		})
	}
}
