package handlers

import (
	"encoding/json"
	"net/http"

	"teachar.in/middleware"
	"teachar.in/models"
)

type createOrderRequest struct {
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	DeliveryAddress string `json:"delivery_address"`
	PaymentMethod   string `json:"payment_method"`
	PaymentStatus   string `json:"payment_status"`
	TransactionID   string `json:"transaction_id"`
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

	if user != nil {
		userID = user.ID
		if customerName == "" {
			customerName = user.Name
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
		CustomerPhone:   req.CustomerPhone,
		DeliveryAddress: req.DeliveryAddress,
		PaymentMethod:   req.PaymentMethod,
		PaymentStatus:   req.PaymentStatus,
		TransactionID:   req.TransactionID,
		Items:           orderItems,
	}

	createdOrder, err := app.OrderService.CreateOrder(r.Context(), order)
	if err != nil {
		app.badRequestError(w, r, err)
		return
	}

	app.writeJSON(w, r, http.StatusCreated, createdOrder, nil)
}
