package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"teachar.in/middleware"
	"teachar.in/models"
	"teachar.in/services"
)

type createOrderRequest struct {
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	OrderType       string `json:"order_type"`
	TableNumber     string `json:"table_number"`
	DeliveryAddress string `json:"delivery_address"`
	PaymentMethod   string `json:"payment_method"`
	PaymentStatus   string `json:"payment_status"`
	TransactionID   string `json:"transaction_id"`
	CouponCode      string `json:"coupon_code"`
	Items           []struct {
		ID       int64   `json:"id"`
		Name     string  `json:"name"`
		Price    float64 `json:"price"`
		Quantity int     `json:"quantity"`
	} `json:"items"`
}

func (app *Application) clientOrdersHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login?redirect=/orders", http.StatusSeeOther)
		return
	}

	orders, err := app.OrderService.GetClientOrders(r.Context(), user.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := models.PageData{
		"Title":  "My Orders",
		"Orders": orders,
	}
	app.render(w, r, http.StatusOK, "client_orders.html", data)
}

func (app *Application) clientAccountHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	orders, _ := app.OrderService.GetClientOrders(r.Context(), user.ID)

	data := models.PageData{
		"Title":       "My Account",
		"User":        user,
		"TotalOrders": len(orders),
	}
	app.render(w, r, http.StatusOK, "client_account.html", data)
}

func (app *Application) apiCreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	user := middleware.GetUserFromContext(r)
	var userID int64 = 0
	var customerName = req.CustomerName
	var customerPhone = req.CustomerPhone

	if user != nil {
		userID = user.ID
		if customerName == "" {
			customerName = user.Name
		}
		if customerPhone == "" {
			customerPhone = user.MobileNumber
		}
	}

	if customerName == "" {
		customerName = "Guest Customer"
	}

	var orderItems []models.OrderItem
	for _, item := range req.Items {
		orderItems = append(orderItems, models.OrderItem{
			MenuItemID: item.ID,
			ItemName:   item.Name,
			Price:      item.Price,
			Quantity:   item.Quantity,
		})
	}

	order := models.Order{
		UserID:          userID,
		CustomerName:    customerName,
		CustomerPhone:   customerPhone,
		OrderType:       req.OrderType,
		TableNumber:     req.TableNumber,
		DeliveryAddress: req.DeliveryAddress,
		PaymentMethod:   req.PaymentMethod,
		PaymentStatus:   req.PaymentStatus,
		TransactionID:   req.TransactionID,
		CouponCode:      req.CouponCode,
		Items:           orderItems,
	}

	createdOrder, err := app.OrderService.CreateOrder(r.Context(), order)
	if err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if app.AuditService != nil {
		details := "Placed Order #" + strconv.FormatInt(createdOrder.ID, 10) + " (" + createdOrder.OrderType + ", Total: ₹" + strconv.FormatFloat(createdOrder.TotalPrice, 'f', 2, 64) + ")"
		app.AuditService.LogEvent(r.Context(), user, "ORDER_CREATED", details, services.GetClientIP(r))
	}

	app.writeJSON(w, r, http.StatusCreated, createdOrder, nil)
}

type submitReviewRequest struct {
	OrderID int64  `json:"order_id"`
	Rating  int    `json:"rating"`
	Review  string `json:"review"`
}

func (app *Application) apiSubmitOrderReviewHandler(w http.ResponseWriter, r *http.Request) {
	var req submitReviewRequest
	if r.Header.Get("Content-Type") == "application/json" {
		json.NewDecoder(r.Body).Decode(&req)
	} else {
		r.ParseForm()
		req.OrderID, _ = strconv.ParseInt(r.FormValue("order_id"), 10, 64)
		req.Rating, _ = strconv.Atoi(r.FormValue("rating"))
		req.Review = r.FormValue("review")
	}

	user := middleware.GetUserFromContext(r)
	var userID int64
	if user != nil {
		userID = user.ID
	}

	if err := app.OrderService.SubmitOrderReview(r.Context(), req.OrderID, userID, req.Rating, req.Review); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if app.AuditService != nil {
		details := "Submitted " + strconv.Itoa(req.Rating) + "-Star review for Order #" + strconv.FormatInt(req.OrderID, 10)
		app.AuditService.LogEvent(r.Context(), user, "REVIEW_SUBMITTED", details, services.GetClientIP(r))
	}

	referer := r.Header.Get("Referer")
	if referer != "" && r.Header.Get("X-Requested-With") == "" {
		http.Redirect(w, r, referer, http.StatusSeeOther)
		return
	}

	app.writeJSON(w, r, http.StatusOK, map[string]string{"message": "Review submitted successfully"}, nil)
}
