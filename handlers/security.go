package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"teachar.in/middleware"
	"teachar.in/models"
	"teachar.in/services"
)

func (app *Application) adminAPIKeysHandler(w http.ResponseWriter, r *http.Request) {
	if app.SecurityService == nil {
		app.serverError(w, r, fmt.Errorf("security service not initialized"))
		return
	}

	keys, err := app.SecurityService.GetAllAPIKeys(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	newKeySecret := r.URL.Query().Get("new_key")

	var activeCount int
	for _, k := range keys {
		if k.IsActive {
			activeCount++
		}
	}

	data := models.PageData{
		"Title":        "API Authentication Keys & Developer Integrations",
		"APIKeys":      keys,
		"ActiveCount":  activeCount,
		"NewKeySecret": newKeySecret,
	}
	app.render(w, r, http.StatusOK, "admin_api_keys.html", data)
}

func (app *Application) adminCreateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	name := r.FormValue("name")
	role := r.FormValue("role")
	expiryDays, _ := strconv.Atoi(r.FormValue("expiry_days"))

	savedKey, rawKeySecret, err := app.SecurityService.GenerateAPIKey(r.Context(), name, role, expiryDays)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	actor := middleware.GetUserFromContext(r)
	if app.AuditService != nil && actor != nil {
		app.AuditService.LogEvent(r.Context(), actor, "API_KEY_GENERATED",
			fmt.Sprintf("Issued API Key '%s' (ID #%d, Role: %s)", savedKey.Name, savedKey.ID, savedKey.Role),
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/api-keys?new_key="+rawKeySecret, http.StatusSeeOther)
}

func (app *Application) adminRevokeAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err := app.SecurityService.RevokeAPIKey(r.Context(), id); err != nil {
		app.serverError(w, r, err)
		return
	}

	actor := middleware.GetUserFromContext(r)
	if app.AuditService != nil && actor != nil {
		app.AuditService.LogEvent(r.Context(), actor, "API_KEY_REVOKED",
			fmt.Sprintf("Revoked API Key #%d", id),
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/api-keys", http.StatusSeeOther)
}
