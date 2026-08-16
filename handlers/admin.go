package handlers

import (
	"net/http"
	"strconv"

	"teachar.in/middleware"
	"teachar.in/models"
	"teachar.in/services"
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

	reasons, _ := app.OrderService.GetCancellationReasons(r.Context())

	data := models.PageData{
		"Title":               "Manage Orders",
		"Orders":              orders,
		"CancellationReasons": reasons,
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
	cancellationReason := r.FormValue("cancellation_reason")
	actor := middleware.GetUserFromContext(r)

	if err := app.OrderService.UpdateOrderStatusWithStaff(r.Context(), id, status, cancellationReason, actor); err != nil {
		orders, _ := app.OrderService.GetAllOrders(r.Context())
		reasons, _ := app.OrderService.GetCancellationReasons(r.Context())
		data := models.PageData{
			"Title":               "Manage Orders",
			"Error":               err.Error(),
			"Orders":              orders,
			"CancellationReasons": reasons,
		}
		app.render(w, r, http.StatusConflict, "admin_orders.html", data)
		return
	}

	if app.AuditService != nil {
		details := "Updated Order #" + strconv.FormatInt(id, 10) + " status to '" + status + "'"
		if cancellationReason != "" {
			details += " [Reason: " + cancellationReason + "]"
		}
		if actor != nil {
			details += " (Handled by: " + actor.Name + ")"
		}
		app.AuditService.LogEvent(r.Context(), actor, "ORDER_STATUS_UPDATED", details, services.GetClientIP(r))
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/admin"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func (app *Application) adminUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := app.AuthService.GetAllUsers(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	reasons, _ := app.OrderService.GetCancellationReasons(r.Context())

	data := models.PageData{
		"Title":               "Staff & Registered Users",
		"Users":               users,
		"CancellationReasons": reasons,
	}
	app.render(w, r, http.StatusOK, "admin_users.html", data)
}

func (app *Application) adminAddCancellationReasonHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	reason := r.FormValue("reason")
	if err := app.OrderService.AddCancellationReason(r.Context(), reason); err != nil {
		app.serverError(w, r, err)
		return
	}

	if app.AuditService != nil {
		actor := middleware.GetUserFromContext(r)
		app.AuditService.LogEvent(r.Context(), actor, "REASON_ADDED", "Added cancellation reason: '"+reason+"'", services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (app *Application) adminDeleteCancellationReasonHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	reason := r.FormValue("reason")
	if err := app.OrderService.DeleteCancellationReason(r.Context(), reason); err != nil {
		app.serverError(w, r, err)
		return
	}

	if app.AuditService != nil {
		actor := middleware.GetUserFromContext(r)
		app.AuditService.LogEvent(r.Context(), actor, "REASON_DELETED", "Deleted cancellation reason: '"+reason+"'", services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (app *Application) adminCreateStaffHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	mobile := r.FormValue("mobile")
	password := r.FormValue("password")
	role := r.FormValue("role")

	if role != "staff" && role != "admin" {
		role = "staff"
	}

	createdUser, err := app.AuthService.CreateUserWithRole(r.Context(), name, email, mobile, password, role)
	if err != nil {
		users, _ := app.AuthService.GetAllUsers(r.Context())
		data := models.PageData{
			"Title": "Staff & Registered Users",
			"Error": err.Error(),
			"Users": users,
		}
		app.render(w, r, http.StatusBadRequest, "admin_users.html", data)
		return
	}

	if app.AuditService != nil {
		actor := middleware.GetUserFromContext(r)
		details := "Created " + role + " account for " + createdUser.Name + " (" + createdUser.Email + ")"
		app.AuditService.LogEvent(r.Context(), actor, "STAFF_CREATED", details, services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (app *Application) adminAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	logs, err := app.AuditService.GetAllLogs(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := models.PageData{
		"Title":     "System Audit Logs",
		"AuditLogs": logs,
	}
	app.render(w, r, http.StatusOK, "admin_audit_logs.html", data)
}

func (app *Application) adminReportsHandler(w http.ResponseWriter, r *http.Request) {
	report, err := app.ReportService.GenerateFinancialReport(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := models.PageData{
		"Title":  "Financial & Auditing Reports",
		"Report": report,
	}
	app.render(w, r, http.StatusOK, "admin_reports.html", data)
}
