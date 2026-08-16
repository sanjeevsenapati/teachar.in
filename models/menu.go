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
	ID                 int64       `json:"id"`
	UserID             int64       `json:"user_id"`
	CustomerName       string      `json:"customer_name"`
	CustomerPhone      string      `json:"customer_phone"`
	OrderType          string      `json:"order_type"`           // "Dine-in", "Takeaway", "Delivery"
	TableNumber        string      `json:"table_number"`         // Required for Dine-in
	DeliveryAddress    string      `json:"delivery_address"`      // Required for Delivery
	Status             string      `json:"status"`                // "Pending", "Preparing", "Ready", "Completed", "Cancelled"
	PaymentMethod      string      `json:"payment_method"`        // "UPI", "Card", "NetBanking", "COD"
	PaymentStatus      string      `json:"payment_status"`        // "Paid", "Pending", "Failed"
	TransactionID      string      `json:"transaction_id"`
	TotalPrice         float64     `json:"total_price"`
	AssignedStaffID    int64       `json:"assigned_staff_id"`    // Staff member who claimed/handled this order
	AssignedStaffName  string      `json:"assigned_staff_name"`  // Staff member's name
	CancellationReason string      `json:"cancellation_reason"`  // Staff cancellation explanation
	Items              []OrderItem `json:"items"`
	CreatedAt          time.Time   `json:"created_at"`
}

// PageData holds the data to be passed to HTML templates.
type PageData map[string]interface{}

// AuditLog represents an auditable system event.
type AuditLog struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	ActorID   int64     `json:"actor_id"`
	ActorName string    `json:"actor_name"`
	ActorRole string    `json:"actor_role"`
	Action    string    `json:"action"` // e.g. "USER_REGISTER", "USER_LOGIN", "STAFF_CREATED", "ORDER_CREATED", "ORDER_STATUS_UPDATED", "MENU_ITEM_ADDED"
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
}

// PaymentMethodReport holds aggregated financials for a payment method.
type PaymentMethodReport struct {
	Method      string  `json:"method"`
	OrderCount  int     `json:"order_count"`
	TotalAmount float64 `json:"total_amount"`
}

// OrderTypeReport holds aggregated financials for an order type.
type OrderTypeReport struct {
	Type        string  `json:"type"`
	OrderCount  int     `json:"order_count"`
	TotalAmount float64 `json:"total_amount"`
}

// CategoryReport holds aggregated sales by menu category.
type CategoryReport struct {
	Category    string  `json:"category"`
	ItemsSold   int     `json:"items_sold"`
	TotalAmount float64 `json:"total_amount"`
}

// TopItemReport holds sales statistics for individual menu items.
type TopItemReport struct {
	ItemName    string  `json:"item_name"`
	Category    string  `json:"category"`
	Quantity    int     `json:"quantity"`
	TotalRevenue float64 `json:"total_revenue"`
}

// FinancialReportData holds complete auditing and revenue metrics.
type FinancialReportData struct {
	GrossRevenue      float64               `json:"gross_revenue"`
	TotalTax          float64               `json:"total_tax"`
	NetRevenue        float64               `json:"net_revenue"`
	TotalOrders       int                   `json:"total_orders"`
	CompletedOrders   int                   `json:"completed_orders"`
	PendingOrders     int                   `json:"pending_orders"`
	CancelledOrders   int                   `json:"cancelled_orders"`
	AverageOrderValue float64               `json:"average_order_value"`
	PaidAmount        float64               `json:"paid_amount"`
	PendingPayment    float64               `json:"pending_payment"`
	PaymentMethods    []PaymentMethodReport `json:"payment_methods"`
	OrderTypes        []OrderTypeReport     `json:"order_types"`
	CategorySales     []CategoryReport      `json:"category_sales"`
	TopSellingItems   []TopItemReport       `json:"top_selling_items"`
}

