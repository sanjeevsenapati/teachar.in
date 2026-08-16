package handlers

import (
	"net/http"
	"strconv"

	"teachar.in/models"
)

func (app *Application) adminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	orders, err := app.OrderService.GetAllOrders(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	menuMap, _ := app.MenuService.GetFullMenu(r.Context())
	totalMenuItems := 0
	for _, items := range menuMap {
		totalMenuItems += len(items)
	}

	var totalRevenue float64
	pendingOrders := 0
	for _, o := range orders {
		if o.Status != "Cancelled" {
			totalRevenue += o.TotalPrice
		}
		if o.Status == "Pending" || o.Status == "Preparing" {
			pendingOrders++
		}
	}

	data := models.PageData{
		"Title":          "Admin Dashboard",
		"TotalRevenue":   totalRevenue,
		"TotalOrders":    len(orders),
		"PendingOrders":  pendingOrders,
		"TotalMenuItems": totalMenuItems,
		"RecentOrders":   orders,
	}
	app.render(w, r, http.StatusOK, "admin_dashboard.html", data)
}

func (app *Application) adminMenuHandler(w http.ResponseWriter, r *http.Request) {
	menuMap, err := app.MenuService.GetFullMenu(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	var flatMenu []models.MenuItem
	for _, items := range menuMap {
		flatMenu = append(flatMenu, items...)
	}

	data := models.PageData{
		"Title": "Manage Menu",
		"Menu":  flatMenu,
	}
	app.render(w, r, http.StatusOK, "admin_menu.html", data)
}

func (app *Application) adminAddMenuItemHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	image := r.FormValue("image")
	if image == "" {
		image = "/static/images/masala-tea.jpg"
	}

	item := models.MenuItem{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Category:    r.FormValue("category"),
		Price:       price,
		Image:       image,
		Available:   true,
	}

	_, err := app.MenuService.CreateItem(r.Context(), item)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
}

func (app *Application) adminEditMenuItemHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	available := r.FormValue("available") == "true"

	item := models.MenuItem{
		ID:          id,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Category:    r.FormValue("category"),
		Price:       price,
		Image:       r.FormValue("image"),
		Available:   available,
	}

	if err := app.MenuService.UpdateItem(r.Context(), item); err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
}

func (app *Application) adminDeleteMenuItemHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err := app.MenuService.DeleteItem(r.Context(), id); err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
}

func (app *Application) adminToggleMenuItemHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err := app.MenuService.ToggleAvailability(r.Context(), id); err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
}

func (app *Application) adminOrdersHandler(w http.ResponseWriter, r *http.Request) {
	orders, err := app.OrderService.GetAllOrders(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := models.PageData{
		"Title":  "Manage Orders",
		"Orders": orders,
	}
	app.render(w, r, http.StatusOK, "admin_orders.html", data)
}

func (app *Application) adminUpdateOrderStatusHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	status := r.FormValue("status")

	if err := app.OrderService.UpdateOrderStatus(r.Context(), id, status); err != nil {
		app.serverError(w, r, err)
		return
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/admin"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}
