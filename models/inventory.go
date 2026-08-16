package models

import "time"

// InventoryItem represents a raw material, stock consumable, equipment, or furniture asset.
type InventoryItem struct {
	ID            int64     `json:"id"`
	Category      string    `json:"category"`       // "Raw Material", "Consumable", "Equipment", "Furniture"
	ItemName      string    `json:"item_name"`      // e.g. "Assam Tea Leaves", "Espresso Machine", "Wooden Table"
	Unit          string    `json:"unit"`           // "kg", "liters", "units", "packs", "boxes"
	StockQuantity float64   `json:"stock_quantity"` // Current stock on hand
	ReorderLevel  float64   `json:"reorder_level"`  // Threshold for low stock warning
	UnitCost      float64   `json:"unit_cost"`      // Purchase cost per unit
	TotalValue    float64   `json:"total_value"`    // StockQuantity * UnitCost
	Supplier      string    `json:"supplier"`       // Vendor / Supplier name
	SerialNumber  string    `json:"serial_number,omitempty"` // For equipment assets
	Status        string    `json:"status"`         // "In Stock", "Low Stock", "Out of Stock", "Active Asset", "Under Maintenance"
	UpdatedAt     time.Time `json:"updated_at"`
}

// ExpenseEntry represents a cafe operating expenditure, purchase voucher, or overhead payment.
type ExpenseEntry struct {
	ID             int64     `json:"id"`
	Category       string    `json:"category"`        // "Raw Materials", "Equipment", "Furniture", "Rent", "Utilities", "Maintenance", "Transportation", "Miscellaneous"
	Title          string    `json:"title"`           // Description of expense / purchase
	SupplierVendor string    `json:"supplier_vendor"` // Vendor, landlord, utility company, or service provider
	InvoiceNo      string    `json:"invoice_no"`      // Bill / Invoice / Voucher #
	Quantity       float64   `json:"quantity"`        // Quantity purchased (if applicable)
	UnitPrice      float64   `json:"unit_price"`      // Cost per unit (if applicable)
	TotalAmount    float64   `json:"total_amount"`    // Total expense amount paid
	PaymentMethod  string    `json:"payment_method"`  // "Bank Transfer", "UPI", "Cash", "Cheque", "Card"
	ExpenseDate    time.Time `json:"expense_date"`    // Date expense occurred
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// TaxAuditReport holds financial year or periodic tax audit metrics.
type TaxAuditReport struct {
	Period              string         `json:"period"`                // "FY 2025-26", "Today", "Monthly", etc.
	GrossRevenue        float64        `json:"gross_revenue"`         // Sales revenue from customer orders
	COGS                float64        `json:"cogs"`                  // Raw materials & consumables cost
	OperatingExpenses   float64        `json:"operating_expenses"`    // Rent, utilities, maintenance, transport
	TotalCapitalAssets  float64        `json:"total_capital_assets"`  // Total value of equipment & furniture
	TotalExpenditure    float64        `json:"total_expenditure"`     // COGS + OperatingExpenses
	NetOperatingProfit  float64        `json:"net_operating_profit"`  // Revenue - TotalExpenditure
	EstimatedTaxableInc float64        `json:"estimated_taxable_inc"` // Taxable income estimation
	EstimatedGST        float64        `json:"estimated_gst"`         // Estimated GST (5% hospitality standard)
	EstimatedIncomeTax  float64        `json:"estimated_income_tax"`  // Estimated income tax (approx 25%)
	CategoryBreakdown   map[string]float64 `json:"category_breakdown"`
}
