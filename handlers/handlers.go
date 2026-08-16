package handlers

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"teachar.in/config"
	"teachar.in/middleware"
	"teachar.in/models"
	"teachar.in/services"
)

// Application holds the application-wide dependencies.
type Application struct {
	Logger       *slog.Logger
	Config       *config.Config
	MenuService  *services.MenuService
	AuthService  *services.AuthService
	OrderService *services.OrderService
}

// RegisterRoutes sets up all the routes for the application.
func (app *Application) RegisterRoutes(mux *http.ServeMux) {
	mw := middleware.NewManager(app.Logger, app.AuthService)
	chainedHandler := mw.Chain(mw.Recovery, mw.Logging, mw.Security, mw.AuthenticateSession)

	router := http.NewServeMux()

	// Static file server
	fileServer := http.FileServer(http.Dir("./static/"))
	router.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Public page handlers
	router.HandleFunc("GET /{$}", app.homeHandler)
	router.HandleFunc("GET /menu", app.menuHandler)
	router.HandleFunc("GET /about", app.aboutHandler)

	// Auth handlers
	router.HandleFunc("GET /login", app.loginPageHandler)
	router.HandleFunc("POST /login", app.loginSubmitHandler)
	router.HandleFunc("GET /register", app.registerPageHandler)
	router.HandleFunc("POST /register", app.registerSubmitHandler)
	router.HandleFunc("POST /logout", app.logoutHandler)

	// Client handlers
	router.HandleFunc("GET /orders", app.clientOrdersHandler)
	router.HandleFunc("GET /account", app.clientAccountHandler)
	router.HandleFunc("POST /api/orders", app.apiCreateOrderHandler)

	// Admin handlers (protected by RequireAdmin)
	router.Handle("GET /admin", mw.RequireAdmin(http.HandlerFunc(app.adminDashboardHandler)))
	router.Handle("GET /admin/menu", mw.RequireAdmin(http.HandlerFunc(app.adminMenuHandler)))
	router.Handle("POST /admin/menu/add", mw.RequireAdmin(http.HandlerFunc(app.adminAddMenuItemHandler)))
	router.Handle("POST /admin/menu/edit", mw.RequireAdmin(http.HandlerFunc(app.adminEditMenuItemHandler)))
	router.Handle("POST /admin/menu/delete", mw.RequireAdmin(http.HandlerFunc(app.adminDeleteMenuItemHandler)))
	router.Handle("POST /admin/menu/toggle", mw.RequireAdmin(http.HandlerFunc(app.adminToggleMenuItemHandler)))
	router.Handle("GET /admin/orders", mw.RequireAdmin(http.HandlerFunc(app.adminOrdersHandler)))
	router.Handle("POST /admin/orders/status", mw.RequireAdmin(http.HandlerFunc(app.adminUpdateOrderStatusHandler)))

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
		data["IsAdmin"] = (user.Role == "admin")
	} else {
		data["IsAuthenticated"] = false
		data["IsAdmin"] = false
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		app.serverError(w, r, err)
	}
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
