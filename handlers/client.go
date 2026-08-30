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
		ID         int64   `json:"id"`
		MenuItemID int64   `json:"menu_item_id"`
		Name       string  `json:"name"`
		ItemName   string  `json:"item_name"`
		Price      float64 `json:"price"`
		Quantity   int     `json:"quantity"`
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
		itemID := item.MenuItemID
		if itemID == 0 {
			itemID = item.ID
		}
		itemName := item.ItemName
		if itemName == "" {
			itemName = item.Name
		}
		orderItems = append(orderItems, models.OrderItem{
			MenuItemID: itemID,
			ItemName:   itemName,
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

func (app *Application) apiSubmitOrderReviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.methodNotAllowedError(w, r)
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	orderIDStr := r.FormValue("order_id")
	ratingStr := r.FormValue("rating")
	review := r.FormValue("review")

	orderID, _ := strconv.ParseInt(orderIDStr, 10, 64)
	rating, _ := strconv.Atoi(ratingStr)

	if orderID > 0 && rating >= 1 && rating <= 5 {
		_ = app.OrderService.SubmitOrderReview(r.Context(), orderID, user.ID, rating, review)
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

func (app *Application) apiGetClientOrdersStatusHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		app.writeJSON(w, r, http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		}, nil)
		return
	}

	orders, err := app.OrderService.GetClientOrders(r.Context(), user.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.writeJSON(w, r, http.StatusOK, map[string]interface{}{
		"success": true,
		"orders":  orders,
	}, nil)
}
