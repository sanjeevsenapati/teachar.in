package handlers

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"teachar.in/config"
	"teachar.in/middleware"
	"teachar.in/models"
	"teachar.in/services"
)

// Application holds the application-wide dependencies.
type Application struct {
	Logger            *slog.Logger
	Config            *config.Config
	MenuService       *services.MenuService
	AuthService       *services.AuthService
	OrderService      *services.OrderService
	AuditService      *services.AuditService
	ReportService     *services.ReportService
	CouponService     *services.CouponService
	InventoryService  *services.InventoryService
	SecurityService   *services.SecurityService
	MembershipService *services.MembershipService
}

// RegisterRoutes sets up all the routes for the application.
func (app *Application) RegisterRoutes(mux *http.ServeMux) {
	mw := middleware.NewManager(app.Logger, app.AuthService, app.SecurityService)

	var middlewares []middleware.Middleware
	middlewares = append(middlewares, mw.Recovery, mw.Logging, mw.Security)

	if app.Config != nil && app.Config.EnableRateLimit {
		window := time.Duration(app.Config.RateLimitWindowSeconds) * time.Second
		if window <= 0 {
			window = time.Minute
		}
		limit := app.Config.RateLimitRequests
		if limit <= 0 {
			limit = 60
		}
		limiter := middleware.NewRateLimiter(limit, window)
		middlewares = append(middlewares, limiter.LimitMiddleware)
	}

	middlewares = append(middlewares, mw.AuthenticateSession)
	chainedHandler := mw.Chain(middlewares...)

	router := http.NewServeMux()

	// Static file server
	fileServer := http.FileServer(http.Dir("./static/"))
	router.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Public page handlers
	router.HandleFunc("GET /{$}", app.homeHandler)
	router.HandleFunc("GET /menu", app.menuHandler)
	router.HandleFunc("GET /about", app.aboutHandler)
	router.HandleFunc("GET /membership", app.clientMembershipHandler)

	// Auth handlers
	router.HandleFunc("GET /login", app.loginPageHandler)
	router.HandleFunc("POST /login", app.loginSubmitHandler)
	router.HandleFunc("GET /register", app.registerPageHandler)
	router.HandleFunc("POST /register", app.registerSubmitHandler)
	router.HandleFunc("POST /logout", app.logoutHandler)

	// Client dashboard & membership handlers
	router.HandleFunc("GET /account", app.clientAccountHandler)
	router.HandleFunc("POST /api/profile/update", app.apiUpdateProfileHandler)
	router.HandleFunc("GET /orders", app.clientOrdersHandler)
	router.HandleFunc("POST /api/orders", app.apiCreateOrderHandler)
	router.HandleFunc("POST /api/orders/review", app.apiSubmitOrderReviewHandler)
	router.HandleFunc("POST /api/coupons/validate", app.apiValidateCouponHandler)
	router.HandleFunc("POST /api/membership/subscribe", app.apiSubscribeMembershipHandler)
	router.HandleFunc("POST /api/membership/claim-cup", app.apiClaimDailyCupHandler)

	// Staff and Admin shared handlers
	router.Handle("GET /admin", mw.RequireStaffOrAdmin(http.HandlerFunc(app.adminDashboardHandler)))
	router.Handle("GET /admin/menu", mw.RequireAdmin(http.HandlerFunc(app.adminMenuHandler)))
	router.Handle("POST /admin/menu/add", mw.RequireAdmin(http.HandlerFunc(app.adminAddMenuItemHandler)))
	router.Handle("POST /admin/menu/toggle", mw.RequireAdmin(http.HandlerFunc(app.adminToggleMenuItemHandler)))
	router.Handle("GET /admin/orders", mw.RequireStaffOrAdmin(http.HandlerFunc(app.adminOrdersHandler)))
	router.Handle("POST /admin/orders/status", mw.RequireStaffOrAdmin(http.HandlerFunc(app.adminUpdateOrderStatusHandler)))
	router.Handle("POST /admin/orders/assign", mw.RequireAdmin(http.HandlerFunc(app.adminAssignOrderHandler)))
	router.Handle("GET /admin/staff-performance", mw.RequireAdmin(http.HandlerFunc(app.adminStaffPerformanceHandler)))

	// Store Inventory & Expenditure routes
	router.Handle("GET /admin/inventory", mw.RequireAdmin(http.HandlerFunc(app.adminInventoryHandler)))
	router.Handle("POST /admin/inventory/add", mw.RequireAdmin(http.HandlerFunc(app.adminAddInventoryHandler)))
	router.Handle("POST /admin/inventory/delete", mw.RequireAdmin(http.HandlerFunc(app.adminDeleteInventoryHandler)))
	router.Handle("GET /admin/expenses", mw.RequireAdmin(http.HandlerFunc(app.adminExpensesHandler)))
	router.Handle("POST /admin/expenses/add", mw.RequireAdmin(http.HandlerFunc(app.adminAddExpenseHandler)))
	router.Handle("POST /admin/expenses/delete", mw.RequireAdmin(http.HandlerFunc(app.adminDeleteExpenseHandler)))
	router.Handle("GET /admin/inventory/export", mw.RequireAdmin(http.HandlerFunc(app.adminExportInventoryHandler)))
	router.Handle("GET /admin/cafe-settings", mw.RequireAdmin(http.HandlerFunc(app.adminCafeSettingsHandler)))

	// Superadmin & Admin membership handlers
	router.Handle("GET /admin/users", mw.RequireSuperadmin(http.HandlerFunc(app.adminUsersHandler)))
	router.Handle("POST /admin/users/create", mw.RequireSuperadmin(http.HandlerFunc(app.adminCreateStaffHandler)))
	router.Handle("POST /admin/users/create-staff", mw.RequireSuperadmin(http.HandlerFunc(app.adminCreateStaffHandler)))
	router.Handle("POST /admin/cancellation-reasons/add", mw.RequireSuperadmin(http.HandlerFunc(app.adminAddCancellationReasonHandler)))
	router.Handle("POST /admin/cancellation-reasons/delete", mw.RequireSuperadmin(http.HandlerFunc(app.adminDeleteCancellationReasonHandler)))
	router.Handle("GET /admin/audit-logs", mw.RequireSuperadmin(http.HandlerFunc(app.adminAuditLogsHandler)))
	router.Handle("GET /admin/reports", mw.RequireAdmin(http.HandlerFunc(app.adminReportsHandler)))
	router.Handle("GET /admin/reports/export", mw.RequireAdmin(http.HandlerFunc(app.adminReportsExportHandler)))
	router.Handle("GET /admin/coupons", mw.RequireSuperadmin(http.HandlerFunc(app.adminCouponsHandler)))
	router.Handle("POST /admin/coupons/create", mw.RequireSuperadmin(http.HandlerFunc(app.adminCreateCouponHandler)))
	router.Handle("POST /admin/coupons/delete", mw.RequireSuperadmin(http.HandlerFunc(app.adminDeleteCouponHandler)))
	router.Handle("GET /admin/api-keys", mw.RequireSuperadmin(http.HandlerFunc(app.adminAPIKeysHandler)))
	router.Handle("POST /admin/api-keys/create", mw.RequireSuperadmin(http.HandlerFunc(app.adminCreateAPIKeyHandler)))
	router.Handle("POST /admin/api-keys/revoke", mw.RequireSuperadmin(http.HandlerFunc(app.adminRevokeAPIKeyHandler)))
	router.Handle("GET /admin/memberships", mw.RequireAdmin(http.HandlerFunc(app.adminMembershipsHandler)))
	router.Handle("POST /admin/memberships/grant", mw.RequireAdmin(http.HandlerFunc(app.adminGrantMembershipHandler)))
	router.Handle("POST /admin/memberships/cancel", mw.RequireAdmin(http.HandlerFunc(app.adminCancelMembershipHandler)))

	// API status endpoints
	router.HandleFunc("GET /api/status", app.apiStatusHandler)
	router.HandleFunc("GET /health", app.healthCheckHandler)
	router.HandleFunc("GET /api/menu", app.apiGetMenuHandler)
	router.HandleFunc("GET /api/menu/{id}", app.apiGetMenuItemHandler)

	mux.Handle("/", chainedHandler(router))
}

// render is a helper for rendering HTML templates.
func (app *Application) render(w http.ResponseWriter, r *http.Request, status int, page string, data models.PageData) {
	templateDir := "./templates"
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		if _, err := os.Stat("../templates"); err == nil {
			templateDir = "../templates"
		}
	}
	ts, err := template.ParseFiles(filepath.Join(templateDir, "layout.html"), filepath.Join(templateDir, page))
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if data == nil {
		data = models.PageData{}
	}

	data["CurrentYear"] = 2026
	if user := middleware.GetUserFromContext(r); user != nil {
		data["User"] = user
		data["IsAuthenticated"] = true
		data["IsSuperadmin"] = (user.Role == "superadmin")
		data["IsAdmin"] = (user.Role == "admin" || user.Role == "superadmin")
		data["IsStaff"] = (user.Role == "staff")
		data["CanManageOrders"] = (user.Role == "staff" || user.Role == "admin" || user.Role == "superadmin")
	} else {
		data["IsAuthenticated"] = false
		data["IsSuperadmin"] = false
		data["IsAdmin"] = false
		data["IsStaff"] = false
		data["CanManageOrders"] = false
	}

	buf := new(bytes.Buffer)
	err = ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

// writeJSON is a helper for sending JSON responses.
func (app *Application) writeJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}, headers http.Header) {
	js, err := json.Marshal(data)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
}

// errorJSON is a helper for sending JSON error responses.
func (app *Application) errorJSON(w http.ResponseWriter, r *http.Request, status int, message string) {
	type jsonError struct {
		Error string `json:"error"`
	}
	app.writeJSON(w, r, status, jsonError{Error: message}, nil)
}
