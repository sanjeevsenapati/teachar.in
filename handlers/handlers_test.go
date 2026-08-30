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
	"teachar.in/models"
	"teachar.in/repository"
	"teachar.in/services"
)

func setupTestApp(t *testing.T) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg, _ := config.New()
	
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := repository.NewSQLiteRepository(dbPath, tempDir)
	if err != nil {
		t.Fatalf("failed to create test sqlite repo: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	couponService := services.NewCouponService(repo)
	menuService := services.NewMenuService(repo)
	authService := services.NewAuthService(repo)
	membershipService := services.NewMembershipService(repo)
	orderService := services.NewOrderService(repo, couponService, membershipService)
	auditService := services.NewAuditService(repo)
	reportService := services.NewReportService(repo, repo, repo)
	inventoryService := services.NewInventoryService(repo, repo)

	securityService := services.NewSecurityService(repo)

	app := &handlers.Application{
		Logger:            logger,
		Config:            cfg,
		MenuService:       menuService,
		AuthService:       authService,
		OrderService:      orderService,
		AuditService:      auditService,
		ReportService:     reportService,
		CouponService:     couponService,
		InventoryService:  inventoryService,
		SecurityService:   securityService,
		MembershipService: membershipService,
		SettingsRepo:      repo,
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

func TestPublicAndAdminRouteSeparation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg, _ := config.New()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := repository.NewSQLiteRepository(dbPath, tempDir)
	if err != nil {
		t.Fatalf("failed creating repo: %v", err)
	}
	defer repo.Close()

	couponSvc := services.NewCouponService(repo)
	menuSvc := services.NewMenuService(repo)
	authSvc := services.NewAuthService(repo)
	membershipSvc := services.NewMembershipService(repo)
	orderSvc := services.NewOrderService(repo, couponSvc, membershipSvc)
	auditSvc := services.NewAuditService(repo)
	reportSvc := services.NewReportService(repo, repo, repo)
	inventorySvc := services.NewInventoryService(repo, repo)
	secSvc := services.NewSecurityService(repo)

	app := &handlers.Application{
		Logger:            logger,
		Config:            cfg,
		MenuService:       menuSvc,
		AuthService:       authSvc,
		OrderService:      orderSvc,
		AuditService:      auditSvc,
		ReportService:     reportSvc,
		CouponService:     couponSvc,
		InventoryService:  inventorySvc,
		SecurityService:   secSvc,
		MembershipService: membershipSvc,
		SettingsRepo:      repo,
	}

	publicMux := http.NewServeMux()
	app.RegisterPublicRoutes(publicMux)

	adminMux := http.NewServeMux()
	app.RegisterAdminRoutes(adminMux)

	t.Run("PublicApp_HasCustomerRoutes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/menu", nil)
		rec := httptest.NewRecorder()
		publicMux.ServeHTTP(rec, req)
		if rec.Result().StatusCode != http.StatusOK {
			t.Errorf("expected public app to serve /menu, got %d", rec.Result().StatusCode)
		}
	})

	t.Run("PublicApp_BlocksAdminRoutes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		rec := httptest.NewRecorder()
		publicMux.ServeHTTP(rec, req)
		if rec.Result().StatusCode != http.StatusNotFound {
			t.Errorf("expected public app to return 404 for /admin, got %d", rec.Result().StatusCode)
		}
	})

	t.Run("AdminPortal_ServesAdminRoutes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		rec := httptest.NewRecorder()
		adminMux.ServeHTTP(rec, req)
		// Unauthenticated should redirect to /login or return 303/302
		status := rec.Result().StatusCode
		if status != http.StatusSeeOther && status != http.StatusFound && status != http.StatusOK {
			t.Errorf("expected admin portal to handle /admin, got %d", status)
		}
	})
}

func TestOrderStatusUpdateAndPolling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg, _ := config.New()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := repository.NewSQLiteRepository(dbPath, tempDir)
	if err != nil {
		t.Fatalf("failed creating repo: %v", err)
	}
	defer repo.Close()

	couponSvc := services.NewCouponService(repo)
	menuSvc := services.NewMenuService(repo)
	authSvc := services.NewAuthService(repo)
	membershipSvc := services.NewMembershipService(repo)
	orderSvc := services.NewOrderService(repo, couponSvc, membershipSvc)
	auditSvc := services.NewAuditService(repo)
	reportSvc := services.NewReportService(repo, repo, repo)
	inventorySvc := services.NewInventoryService(repo, repo)
	secSvc := services.NewSecurityService(repo)

	app := &handlers.Application{
		Logger:            logger,
		Config:            cfg,
		MenuService:       menuSvc,
		AuthService:       authSvc,
		OrderService:      orderSvc,
		AuditService:      auditSvc,
		ReportService:     reportSvc,
		CouponService:     couponSvc,
		InventoryService:  inventorySvc,
		SecurityService:   secSvc,
		MembershipService: membershipSvc,
		SettingsRepo:      repo,
	}

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	// Create user
	user, err := authSvc.RegisterUser(t.Context(), "Test User", "testuser@teachar.in", "9876543210", "Password123!")
	if err != nil {
		t.Fatalf("failed creating user: %v", err)
	}

	// Create order
	ord, err := orderSvc.CreateOrder(t.Context(), models.Order{
		UserID:        user.ID,
		CustomerName:  user.Name,
		CustomerPhone: user.MobileNumber,
		OrderType:     "Dine-in",
		TableNumber:   "4",
		PaymentMethod: "UPI",
		PaymentStatus: "Paid",
		Items: []models.OrderItem{
			{MenuItemID: 1, ItemName: "Masala Chai", Quantity: 2, Price: 30.0},
		},
	})
	if err != nil {
		t.Fatalf("failed creating order: %v", err)
	}

	if ord.Status != "Pending" {
		t.Fatalf("expected initial order status Pending, got %s", ord.Status)
	}

	// Admin actor updates order status to "Preparing"
	adminUser := &models.User{ID: 100, Name: "Admin User", Role: "admin"}
	err = orderSvc.UpdateOrderStatusWithStaff(t.Context(), ord.ID, "Preparing", "", adminUser)
	if err != nil {
		t.Fatalf("UpdateOrderStatusWithStaff failed: %v", err)
	}

	// Verify status updated in DB
	updatedOrd, err := repo.GetOrderByID(t.Context(), ord.ID)
	if err != nil {
		t.Fatalf("failed getting order: %v", err)
	}
	if updatedOrd.Status != "Preparing" {
		t.Errorf("expected status Preparing, got %s", updatedOrd.Status)
	}
	if updatedOrd.AssignedStaffID != adminUser.ID {
		t.Errorf("expected assigned staff ID %d, got %d", adminUser.ID, updatedOrd.AssignedStaffID)
	}
}


