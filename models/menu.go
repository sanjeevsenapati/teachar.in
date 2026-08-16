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
	Address      string    `json:"address"`
	Avatar       string    `json:"avatar"`
	PasswordHash string    `json:"password_hash"`
	Salt         string    `json:"salt"`
	Role         string    `json:"role"` // "client", "admin", "superadmin", "staff"
	IsLocked     bool      `json:"is_locked"`
	Status       string    `json:"status"` // "Active", "Locked", "Disabled"
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

// Coupon represents a single-use offer discount coupon.
type Coupon struct {
	ID             int64      `json:"id"`
	Code           string     `json:"code"`             // Unique coupon code, e.g. "WELCOME50"
	DiscountType   string     `json:"discount_type"`    // "flat" or "percentage"
	DiscountValue  float64    `json:"discount_value"`   // Amount in ₹ or % percentage value
	MinOrderAmount float64    `json:"min_order_amount"` // Optional minimum order subtotal
	ExpiryDate     time.Time  `json:"expiry_date"`      // Expiration timestamp
	IsUsed         bool       `json:"is_used"`          // Single-use flag
	UsedAt         *time.Time `json:"used_at,omitempty"`
	UsedByOrderID  int64      `json:"used_by_order_id,omitempty"`
	CreatedBy      string     `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
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
	SubtotalPrice      float64     `json:"subtotal_price"`
	CouponCode         string      `json:"coupon_code,omitempty"`
	DiscountAmount     float64     `json:"discount_amount,omitempty"`
	SubscriberDiscount float64     `json:"subscriber_discount,omitempty"`
	SubscriberTierName string      `json:"subscriber_tier_name,omitempty"`
	TotalPrice         float64     `json:"total_price"`
	AssignedStaffID    int64       `json:"assigned_staff_id"`    // Staff member who claimed/handled this order
	AssignedStaffName  string      `json:"assigned_staff_name"`  // Staff member's name
	AssignedBy         string      `json:"assigned_by,omitempty"` // "Admin", "Super Admin", or "Self Claimed"
	AssignedAt         *time.Time  `json:"assigned_at,omitempty"`
	CompletedAt        *time.Time  `json:"completed_at,omitempty"`
	EstimatedMinutes   int         `json:"estimated_minutes,omitempty"`   // Staff target prep time in minutes (e.g. 15, 20, 30)
	FulfillmentMinutes int         `json:"fulfillment_minutes,omitempty"` // Actual duration in minutes to complete
	CancellationReason string      `json:"cancellation_reason"`  // Staff cancellation explanation
	Rating             int         `json:"rating"`               // 1 to 5 stars customer rating
	Review             string      `json:"review"`               // Customer feedback comment
	Items              []OrderItem `json:"items"`
	CreatedAt          time.Time   `json:"created_at"`
}

// StaffPerformanceReport holds aggregated performance metrics for a staff member.
type StaffPerformanceReport struct {
	StaffID               int64   `json:"staff_id"`
	StaffName             string  `json:"staff_name"`
	StaffEmail            string  `json:"staff_email"`
	TotalAssignedOrders   int     `json:"total_assigned_orders"`
	CompletedOrders       int     `json:"completed_orders"`
	OnTimeOrders          int     `json:"on_time_orders"`          // Completed in <= 20 minutes
	OverdueOrders         int     `json:"overdue_orders"`         // Completed in > 20 minutes
	OnTimeRate            float64 `json:"on_time_rate"`           // On-time percentage (0 - 100%)
	AvgFulfillmentMinutes float64 `json:"avg_fulfillment_minutes"` // Average duration in minutes
	TotalRatings          int     `json:"total_ratings"`          // Count of customer 1-5 star reviews
	AvgRating             float64 `json:"avg_rating"`             // Average star rating out of 5
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

// HourlyPoint represents aggregated time-series metrics for an individual hour.
type HourlyPoint struct {
	Hour       int     `json:"hour"`        // 0 to 23
	Label      string  `json:"label"`       // e.g. "08:00 AM", "01:00 PM"
	OrderCount int     `json:"order_count"`
	Revenue    float64 `json:"revenue"`
	IsPeak     bool    `json:"is_peak"`
	IsSlow     bool    `json:"is_slow"`
}

// ReportFilter represents user-selected parameters for dynamic sales & financial reporting.
type ReportFilter struct {
	Period            string `json:"period"`             // "today", "yesterday", "weekly", "monthly", "quarterly", "yearly", "custom"
	StartDateStr      string `json:"start_date_str"`     // "YYYY-MM-DD"
	EndDateStr        string `json:"end_date_str"`       // "YYYY-MM-DD"
	FulfillmentMethod string `json:"fulfillment_method"` // "all", "dine-in", "takeaway", "delivery"
	PaymentMethod     string `json:"payment_method"`     // "all", "upi", "card", "netbanking", "cod"
	OrderStatus       string `json:"order_status"`       // "all", "completed", "pending", "cancelled"
	Category          string `json:"category"`           // "all", "tea", "coffee", "snacks", "beverages"
	SearchQuery       string `json:"search_query"`       // Item name or order ID search
}

// FinancialReportData holds complete auditing, period-filtered, and time-series metrics.
type FinancialReportData struct {
	Period            string                `json:"period"`            // "today", "daily", "weekly", "monthly", "yearly", "custom"
	PeriodLabel       string                `json:"period_label"`      // e.g. "Today (Live Data)", "Custom Range (Aug 01 - Aug 16)"
	Filter            ReportFilter          `json:"filter"`
	StartDate         time.Time             `json:"start_date"`
	EndDate           time.Time             `json:"end_date"`
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
	HourlyAnalytics   []HourlyPoint         `json:"hourly_analytics"`  // 24-hour time series
	PeakRushHour      string                `json:"peak_rush_hour"`    // e.g. "01:00 PM - 02:00 PM (12 Orders, ₹850.00)"
	SlowestHour       string                `json:"slowest_hour"`       // e.g. "04:00 PM - 05:00 PM (1 Order, ₹45.00)"
	MarketingAdvice   string                `json:"marketing_advice"`   // Data-driven marketing recommendation
	PaymentMethods    []PaymentMethodReport `json:"payment_methods"`
	OrderTypes        []OrderTypeReport     `json:"order_types"`
	CategorySales     []CategoryReport      `json:"category_sales"`
	TopSellingItems   []TopItemReport       `json:"top_selling_items"`
}

