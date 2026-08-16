package models

import "time"

// MenuItem represents a product in the menu.
type MenuItem struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Image       string  `json:"image"` // URL to the image, e.g., "/static/images/masala-tea.jpg"
	Available   bool    `json:"available"`
}

// User represents a registered user (client or admin).
type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	MobileNumber string    `json:"mobile_number"`
	PasswordHash string    `json:"password_hash"`
	Salt         string    `json:"salt"`
	Role         string    `json:"role"` // "client" or "admin"
	CreatedAt    time.Time `json:"created_at"`
}

// Session represents an active login session.
type Session struct {
	ID        string    `json:"id"` // Random token
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// OrderItem represents a single item in an order.
type OrderItem struct {
	ID         int64   `json:"id"`
	OrderID    int64   `json:"order_id"`
	MenuItemID int64   `json:"menu_item_id"`
	ItemName   string  `json:"item_name"`
	Quantity   int     `json:"quantity"`
	Price      float64 `json:"price"`
}

// Order represents a customer order.
type Order struct {
	ID              int64       `json:"id"`
	UserID          int64       `json:"user_id"`
	CustomerName    string      `json:"customer_name"`
	CustomerPhone   string      `json:"customer_phone"`
	OrderType       string      `json:"order_type"`    // "Dine-in", "Takeaway", "Delivery"
	TableNumber     string      `json:"table_number"`  // Required for Dine-in
	DeliveryAddress string      `json:"delivery_address"` // Required for Delivery
	Status          string      `json:"status"` // "Pending", "Preparing", "Ready", "Completed", "Cancelled"
	PaymentMethod   string      `json:"payment_method"` // "UPI", "Card", "NetBanking", "COD"
	PaymentStatus   string      `json:"payment_status"` // "Paid", "Pending", "Failed"
	TransactionID   string      `json:"transaction_id"`
	TotalPrice      float64     `json:"total_price"`
	Items           []OrderItem `json:"items"`
	CreatedAt       time.Time   `json:"created_at"`
}

// PageData holds the data to be passed to HTML templates.
type PageData map[string]interface{}

