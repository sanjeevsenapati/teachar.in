package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	var staffList []models.User
	if app.AuthService != nil {
		allUsers, _ := app.AuthService.GetAllUsers(r.Context())
		for _, u := range allUsers {
			if u.Role == "staff" {
				staffList = append(staffList, u)
			}
		}
	}

	data := models.PageData{
		"Title":               "Manage Orders & Staff Fulfillment",
		"Orders":              orders,
		"StaffList":           staffList,
		"CancellationReasons": reasons,
	}
	app.render(w, r, http.StatusOK, "admin_orders.html", data)
}

func (app *Application) adminAssignOrderHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	orderIDStr := r.FormValue("order_id")
	staffIDStr := r.FormValue("staff_id")
	estMinsStr := r.FormValue("estimated_minutes")

	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		app.badRequestError(w, r, fmt.Errorf("invalid order ID"))
		return
	}

	staffID, err := strconv.ParseInt(staffIDStr, 10, 64)
	if err != nil {
		app.badRequestError(w, r, fmt.Errorf("invalid staff ID"))
		return
	}

	estimatedMinutes, _ := strconv.Atoi(estMinsStr)
	if estimatedMinutes <= 0 {
		estimatedMinutes = 20
	}

	staffUser, err := app.AuthService.GetUserByID(r.Context(), staffID)
	if err != nil {
		app.badRequestError(w, r, fmt.Errorf("selected staff member not found"))
		return
	}

	actor := middleware.GetUserFromContext(r)
	assignedBy := "Admin"
	if actor != nil {
		if actor.Role == "superadmin" {
			assignedBy = "Super Admin"
		} else if actor.Role == "admin" {
			assignedBy = "Admin"
		}
	}

	if err := app.OrderService.AssignOrderToStaff(r.Context(), orderID, staffUser, assignedBy, estimatedMinutes); err != nil {
		orders, _ := app.OrderService.GetAllOrders(r.Context())
		reasons, _ := app.OrderService.GetCancellationReasons(r.Context())
		allUsers, _ := app.AuthService.GetAllUsers(r.Context())
		var staffList []models.User
		for _, u := range allUsers {
			if u.Role == "staff" {
				staffList = append(staffList, u)
			}
		}
		data := models.PageData{
			"Title":               "Manage Orders",
			"Error":               err.Error(),
			"Orders":              orders,
			"StaffList":           staffList,
			"CancellationReasons": reasons,
		}
		app.render(w, r, http.StatusConflict, "admin_orders.html", data)
		return
	}

	if app.AuditService != nil && actor != nil {
		app.AuditService.LogEvent(r.Context(), actor, "ORDER_ASSIGNED_TO_STAFF",
			fmt.Sprintf("%s assigned Order #%d to staff %s (ID: %d)", assignedBy, orderID, staffUser.Name, staffUser.ID),
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
}

func (app *Application) adminStaffPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	performanceData, err := app.ReportService.GetStaffPerformanceReport(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := models.PageData{
		"Title":            "Staff Performance & Client Satisfaction Analytics",
		"StaffPerformance": performanceData,
	}
	app.render(w, r, http.StatusOK, "admin_staff_performance.html", data)
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
		allUsers, _ := app.AuthService.GetAllUsers(r.Context())
		var staffList []models.User
		for _, u := range allUsers {
			if u.Role == "staff" {
				staffList = append(staffList, u)
			}
		}
		data := models.PageData{
			"Title":               "Manage Orders",
			"Error":               err.Error(),
			"Orders":              orders,
			"StaffList":           staffList,
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

type customerUserViewModel struct {
	User             models.User
	TotalOrders      int
	ActivePassName   string
	ActivePassID     string
	DailyCupsClaimed int
}

type staffUserViewModel struct {
	User               models.User
	TotalOrdersHandled int
}

func (app *Application) adminUsersHandler(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")
	if tab != "customers" {
		tab = "staff"
	}

	users, err := app.AuthService.GetAllUsers(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	allOrders, _ := app.OrderService.GetAllOrders(r.Context())
	customerOrdersCount := make(map[int64]int)
	staffOrdersCount := make(map[string]int)

	for _, o := range allOrders {
		if o.UserID > 0 {
			customerOrdersCount[o.UserID]++
		}
		if o.AssignedStaffName != "" {
			staffOrdersCount[o.AssignedStaffName]++
		}
	}

	var activeSubs []models.UserSubscription
	if app.MembershipService != nil {
		activeSubs, _ = app.MembershipService.GetAllSubscriptions(r.Context())
	}
	userSubMap := make(map[int64]*models.UserSubscription)
	activeMemberCount := 0

	for i := range activeSubs {
		sub := activeSubs[i]
		if sub.Status == "Active" {
			userSubMap[sub.UserID] = &sub
			activeMemberCount++
		}
	}

	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))

	var staffList []staffUserViewModel
	var customerList []customerUserViewModel

	for _, u := range users {
		if u.Role == "staff" || u.Role == "admin" || u.Role == "superadmin" {
			handledCount := staffOrdersCount[u.Name]
			staffList = append(staffList, staffUserViewModel{
				User:               u,
				TotalOrdersHandled: handledCount,
			})
		} else {
			ordersCount := customerOrdersCount[u.ID]
			passName := "Standard Customer"
			passID := ""
			dailyCups := 0

			if sub, exists := userSubMap[u.ID]; exists {
				passName = sub.TierName
				passID = sub.TierID
				dailyCups = sub.CupsClaimedToday
			}

			if q != "" {
				searchTarget := strings.ToLower(fmt.Sprintf("%s %s %s #cst-%d %s %s", u.Name, u.Email, u.MobileNumber, u.ID, u.Address, passName))
				if !strings.Contains(searchTarget, q) {
					continue
				}
			}

			customerList = append(customerList, customerUserViewModel{
				User:             u,
				TotalOrders:      ordersCount,
				ActivePassName:   passName,
				ActivePassID:     passID,
				DailyCupsClaimed: dailyCups,
			})
		}
	}

	reasons, _ := app.OrderService.GetCancellationReasons(r.Context())

	data := models.PageData{
		"Title":               "Account Management Portal",
		"ActiveTab":           tab,
		"SearchQuery":         q,
		"StaffList":           staffList,
		"CustomerList":        customerList,
		"TotalStaffCount":     len(staffList),
		"TotalCustomerCount":  len(customerList),
		"ActiveMemberCount":   activeMemberCount,
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

	http.Redirect(w, r, "/admin/cafe-settings", http.StatusSeeOther)
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

	http.Redirect(w, r, "/admin/cafe-settings", http.StatusSeeOther)
}

func (app *Application) adminCafeSettingsHandler(w http.ResponseWriter, r *http.Request) {
	reasons, _ := app.OrderService.GetCancellationReasons(r.Context())

	data := models.PageData{
		"Title":               "Daily Cafe Settings",
		"CancellationReasons": reasons,
		"StoreName":           "TEACHAR Flagship Cafe Sanctuary",
		"StoreAddress":        "42 Chai Galleria, MG Road, Tech Hub District, Bangalore, 560001",
		"BrewingHours":        "6:00 AM – 11:30 PM",
		"StorePhone":          "+91 98765 43210",
		"CurrencySymbol":      "₹",
	}
	app.render(w, r, http.StatusOK, "admin_cafe_settings.html", data)
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

func parseReportFilter(r *http.Request) models.ReportFilter {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "today"
	}
	return models.ReportFilter{
		Period:            period,
		StartDateStr:      r.URL.Query().Get("start_date"),
		EndDateStr:        r.URL.Query().Get("end_date"),
		FulfillmentMethod: r.URL.Query().Get("fulfillment_method"),
		PaymentMethod:     r.URL.Query().Get("payment_method"),
		OrderStatus:       r.URL.Query().Get("order_status"),
		Category:          r.URL.Query().Get("category"),
		SearchQuery:       r.URL.Query().Get("q"),
	}
}

func (app *Application) adminReportsHandler(w http.ResponseWriter, r *http.Request) {
	filter := parseReportFilter(r)

	report, err := app.ReportService.GenerateFilteredFinancialReport(r.Context(), filter)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := models.PageData{
		"Title":         "Executive Financial & Time-Series Reports",
		"Report":        report,
		"CurrentPeriod": report.Period,
		"Filter":        report.Filter,
	}
	app.render(w, r, http.StatusOK, "admin_reports.html", data)
}

func (app *Application) adminReportsExportHandler(w http.ResponseWriter, r *http.Request) {
	filter := parseReportFilter(r)

	report, err := app.ReportService.GenerateFilteredFinancialReport(r.Context(), filter)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	filename := fmt.Sprintf("teachar_financial_report_%s.csv", report.Period)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(w)

	// Header / Title
	_ = writer.Write([]string{"TEACHAR.in Executive Financial Report"})
	_ = writer.Write([]string{"Report Period", report.PeriodLabel})
	_ = writer.Write([]string{"Date Generated", time.Now().Format("2006-01-02 15:04:05")})
	_ = writer.Write([]string{})

	// Executive Summary Section
	_ = writer.Write([]string{"--- EXECUTIVE FINANCIAL SUMMARY ---"})
	_ = writer.Write([]string{"Metric", "Value"})
	_ = writer.Write([]string{"Gross Revenue (INR)", fmt.Sprintf("%.2f", report.GrossRevenue)})
	_ = writer.Write([]string{"Total GST Tax Collected (5%)", fmt.Sprintf("%.2f", report.TotalTax)})
	_ = writer.Write([]string{"Net Revenue (INR)", fmt.Sprintf("%.2f", report.NetRevenue)})
	_ = writer.Write([]string{"Average Order Value (INR)", fmt.Sprintf("%.2f", report.AverageOrderValue)})
	_ = writer.Write([]string{"Total Orders Count", strconv.Itoa(report.TotalOrders)})
	_ = writer.Write([]string{"Completed Orders", strconv.Itoa(report.CompletedOrders)})
	_ = writer.Write([]string{"Cancelled Orders", strconv.Itoa(report.CancelledOrders)})
	_ = writer.Write([]string{"Total Paid Amount (INR)", fmt.Sprintf("%.2f", report.PaidAmount)})
	_ = writer.Write([]string{"Pending Payments (INR)", fmt.Sprintf("%.2f", report.PendingPayment)})
	_ = writer.Write([]string{"Peak Rush Hour", report.PeakRushHour})
	_ = writer.Write([]string{"Slowest Hour", report.SlowestHour})
	_ = writer.Write([]string{})

	// Revenue by Payment Method
	_ = writer.Write([]string{"--- REVENUE BY PAYMENT METHOD ---"})
	_ = writer.Write([]string{"Payment Method", "Order Count", "Total Revenue (INR)"})
	for _, pm := range report.PaymentMethods {
		_ = writer.Write([]string{pm.Method, strconv.Itoa(pm.OrderCount), fmt.Sprintf("%.2f", pm.TotalAmount)})
	}
	_ = writer.Write([]string{})

	// Revenue by Order Type
	_ = writer.Write([]string{"--- REVENUE BY ORDER TYPE ---"})
	_ = writer.Write([]string{"Order Type", "Order Count", "Total Revenue (INR)"})
	for _, ot := range report.OrderTypes {
		_ = writer.Write([]string{ot.Type, strconv.Itoa(ot.OrderCount), fmt.Sprintf("%.2f", ot.TotalAmount)})
	}
	_ = writer.Write([]string{})

	// Top Selling Menu Items
	_ = writer.Write([]string{"--- TOP SELLING MENU ITEMS ---"})
	_ = writer.Write([]string{"Item Name", "Category", "Units Sold", "Total Revenue (INR)"})
	for _, item := range report.TopSellingItems {
		_ = writer.Write([]string{item.ItemName, item.Category, strconv.Itoa(item.Quantity), fmt.Sprintf("%.2f", item.TotalRevenue)})
	}

	writer.Flush()
}

func (app *Application) adminCouponsHandler(w http.ResponseWriter, r *http.Request) {
	coupons, err := app.CouponService.GetAllCoupons(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	now := time.Now()
	var activeCount, redeemedCount, expiredCount int
	for _, c := range coupons {
		if c.IsUsed {
			redeemedCount++
		} else if c.ExpiryDate.Before(now) {
			expiredCount++
		} else {
			activeCount++
		}
	}

	data := models.PageData{
		"Title":         "Offers & Single-Use Coupons",
		"Coupons":       coupons,
		"ActiveCount":   activeCount,
		"RedeemedCount": redeemedCount,
		"ExpiredCount":  expiredCount,
	}
	app.render(w, r, http.StatusOK, "admin_coupons.html", data)
}

func (app *Application) adminCreateCouponHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	code := r.FormValue("code")
	discountType := r.FormValue("discount_type")
	discountValueStr := r.FormValue("discount_value")
	minOrderStr := r.FormValue("min_order_amount")
	expiryStr := r.FormValue("expiry_date")

	var discountValue float64
	fmt.Sscanf(discountValueStr, "%f", &discountValue)

	var minOrder float64
	if minOrderStr != "" {
		fmt.Sscanf(minOrderStr, "%f", &minOrder)
	}

	var expiryDate time.Time
	if expiryStr != "" {
		var err error
		// Try HTML datetime-local format "2006-01-02T15:04" or date format "2006-01-02"
		expiryDate, err = time.Parse("2006-01-02T15:04", expiryStr)
		if err != nil {
			expiryDate, err = time.Parse("2006-01-02", expiryStr)
		}
		if err != nil {
			expiryDate = time.Now().Add(24 * time.Hour)
		}
	} else {
		expiryDate = time.Now().Add(24 * time.Hour)
	}

	actor := middleware.GetUserFromContext(r)
	actorName := "Superadmin"
	if actor != nil {
		actorName = actor.Name
	}

	coupon := models.Coupon{
		Code:           code,
		DiscountType:   discountType,
		DiscountValue:  discountValue,
		MinOrderAmount: minOrder,
		ExpiryDate:     expiryDate,
	}

	_, err := app.CouponService.CreateCoupon(r.Context(), coupon, actorName)
	if err != nil {
		coupons, _ := app.CouponService.GetAllCoupons(r.Context())
		data := models.PageData{
			"Title":   "Offers & Single-Use Coupons",
			"Error":   err.Error(),
			"Coupons": coupons,
		}
		app.render(w, r, http.StatusBadRequest, "admin_coupons.html", data)
		return
	}

	if app.AuditService != nil {
		app.AuditService.LogEvent(r.Context(), actor, "COUPON_CREATED", "Created single-use coupon '"+code+"'", services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/coupons", http.StatusSeeOther)
}

func (app *Application) adminDeleteCouponHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	var id int64
	fmt.Sscanf(r.FormValue("id"), "%d", &id)

	if err := app.CouponService.DeleteCoupon(r.Context(), id); err != nil {
		app.serverError(w, r, err)
		return
	}

	if app.AuditService != nil {
		actor := middleware.GetUserFromContext(r)
		app.AuditService.LogEvent(r.Context(), actor, "COUPON_DELETED", fmt.Sprintf("Deleted coupon ID #%d", id), services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/coupons", http.StatusSeeOther)
}
