package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// healthCheckHandler provides a simple health check endpoint.
func (app *Application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status":      "UP",
		"application": app.Config.AppName,
	}
	app.writeJSON(w, r, http.StatusOK, data, nil)
}

// apiStatusHandler provides a detailed API status endpoint.
func (app *Application) apiStatusHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status":      "UP",
		"application": app.Config.AppName,
		"environment": app.Config.Env,
		"version":     "1.0.0",
	}
	app.writeJSON(w, r, http.StatusOK, data, nil)
}

// apiGetMenuHandler returns the entire menu as JSON.
func (app *Application) apiGetMenuHandler(w http.ResponseWriter, r *http.Request) {
	menu, err := app.MenuService.GetFullMenu(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.writeJSON(w, r, http.StatusOK, menu, nil)
}

// apiGetMenuItemHandler returns a single menu item by its ID.
func (app *Application) apiGetMenuItemHandler(w http.ResponseWriter, r *http.Request) {
	// The new router in Go 1.22 provides path parameters.
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 1 {
		app.notFoundError(w, r)
		return
	}

	item, err := app.MenuService.GetItemByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			app.notFoundError(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.writeJSON(w, r, http.StatusOK, item, nil)
}

// apiValidateCouponHandler validates a coupon code and calculates discounts asynchronously.
func (app *Application) apiValidateCouponHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code     string  `json:"code"`
		Subtotal float64 `json:"subtotal"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		app.writeJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"valid": false,
			"error": "Invalid request body",
		}, nil)
		return
	}

	coupon, disc, finalTotal, err := app.CouponService.ValidateCoupon(r.Context(), input.Code, input.Subtotal)
	if err != nil {
		app.writeJSON(w, r, http.StatusOK, map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		}, nil)
		return
	}

	app.writeJSON(w, r, http.StatusOK, map[string]interface{}{
		"valid":           true,
		"code":            coupon.Code,
		"discount_type":   coupon.DiscountType,
		"discount_value":  coupon.DiscountValue,
		"discount_amount": disc,
		"subtotal":        input.Subtotal,
		"final_total":     finalTotal,
	}, nil)
}
