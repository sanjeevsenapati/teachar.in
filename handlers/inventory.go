package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"teachar.in/middleware"
	"teachar.in/models"
	"teachar.in/services"
)

func (app *Application) adminInventoryHandler(w http.ResponseWriter, r *http.Request) {
	if app.InventoryService == nil {
		app.serverError(w, r, fmt.Errorf("inventory service not initialized"))
		return
	}

	items, err := app.InventoryService.GetAllInventory(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	var totalStockValue float64
	var lowStockCount int
	var assetCount int

	for _, item := range items {
		totalStockValue += item.TotalValue
		if item.Status == "Low Stock" || item.Status == "Out of Stock" {
			lowStockCount++
		}
		if item.Category == "Equipment" || item.Category == "Furniture" {
			assetCount++
		}
	}

	data := models.PageData{
		"Title":           "Store Inventory & Capital Asset Register",
		"Inventory":       items,
		"TotalStockValue": totalStockValue,
		"LowStockCount":   lowStockCount,
		"AssetCount":      assetCount,
	}
	app.render(w, r, http.StatusOK, "admin_inventory.html", data)
}

func (app *Application) adminAddInventoryHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	category := r.FormValue("category")
	itemName := r.FormValue("item_name")
	unit := r.FormValue("unit")
	stockQty, _ := strconv.ParseFloat(r.FormValue("stock_quantity"), 64)
	reorderLvl, _ := strconv.ParseFloat(r.FormValue("reorder_level"), 64)
	unitCost, _ := strconv.ParseFloat(r.FormValue("unit_cost"), 64)
	supplier := r.FormValue("supplier")
	serialNo := r.FormValue("serial_number")

	item := models.InventoryItem{
		ID:            id,
		Category:      category,
		ItemName:      itemName,
		Unit:          unit,
		StockQuantity: stockQty,
		ReorderLevel:  reorderLvl,
		UnitCost:      unitCost,
		Supplier:      supplier,
		SerialNumber:  serialNo,
	}

	saved, err := app.InventoryService.SaveInventoryItem(r.Context(), item)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	actor := middleware.GetUserFromContext(r)
	if app.AuditService != nil && actor != nil {
		app.AuditService.LogEvent(r.Context(), actor, "INVENTORY_ITEM_SAVED",
			fmt.Sprintf("Saved Inventory Item '%s' (Category: %s, Qty: %.2f %s)", saved.ItemName, saved.Category, saved.StockQuantity, saved.Unit),
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/inventory", http.StatusSeeOther)
}

func (app *Application) adminDeleteInventoryHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err := app.InventoryService.DeleteInventoryItem(r.Context(), id); err != nil {
		app.serverError(w, r, err)
		return
	}

	actor := middleware.GetUserFromContext(r)
	if app.AuditService != nil && actor != nil {
		app.AuditService.LogEvent(r.Context(), actor, "INVENTORY_ITEM_DELETED",
			fmt.Sprintf("Deleted Inventory Item #%d", id),
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/inventory", http.StatusSeeOther)
}

func (app *Application) adminExpensesHandler(w http.ResponseWriter, r *http.Request) {
	if app.InventoryService == nil {
		app.serverError(w, r, fmt.Errorf("inventory service not initialized"))
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "fy"
	}

	expenses, err := app.InventoryService.GetAllExpenses(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	auditReport, err := app.InventoryService.GenerateTaxAuditReport(r.Context(), period)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := models.PageData{
		"Title":       "Operating Expenditure Log & Financial Year Tax Audit",
		"Expenses":    expenses,
		"AuditReport": auditReport,
		"Period":      period,
	}
	app.render(w, r, http.StatusOK, "admin_expenses.html", data)
}

func (app *Application) adminAddExpenseHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	category := r.FormValue("category")
	title := r.FormValue("title")
	vendor := r.FormValue("supplier_vendor")
	invoiceNo := r.FormValue("invoice_no")
	qty, _ := strconv.ParseFloat(r.FormValue("quantity"), 64)
	unitPrice, _ := strconv.ParseFloat(r.FormValue("unit_price"), 64)
	totalAmount, _ := strconv.ParseFloat(r.FormValue("total_amount"), 64)
	paymentMethod := r.FormValue("payment_method")
	expenseDateStr := r.FormValue("expense_date")
	notes := r.FormValue("notes")

	var expenseDate time.Time
	if expenseDateStr != "" {
		expenseDate, _ = time.Parse("2006-01-02", expenseDateStr)
	}
	if expenseDate.IsZero() {
		expenseDate = time.Now()
	}

	expense := models.ExpenseEntry{
		ID:             id,
		Category:       category,
		Title:          title,
		SupplierVendor: vendor,
		InvoiceNo:      invoiceNo,
		Quantity:       qty,
		UnitPrice:      unitPrice,
		TotalAmount:    totalAmount,
		PaymentMethod:  paymentMethod,
		ExpenseDate:    expenseDate,
		Notes:          notes,
	}

	saved, err := app.InventoryService.SaveExpense(r.Context(), expense)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	actor := middleware.GetUserFromContext(r)
	if app.AuditService != nil && actor != nil {
		app.AuditService.LogEvent(r.Context(), actor, "EXPENSE_VOUCHER_SAVED",
			fmt.Sprintf("Logged Expense '%s' (Category: %s, Amount: ₹%.2f)", saved.Title, saved.Category, saved.TotalAmount),
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/expenses", http.StatusSeeOther)
}

func (app *Application) adminDeleteExpenseHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err := app.InventoryService.DeleteExpense(r.Context(), id); err != nil {
		app.serverError(w, r, err)
		return
	}

	actor := middleware.GetUserFromContext(r)
	if app.AuditService != nil && actor != nil {
		app.AuditService.LogEvent(r.Context(), actor, "EXPENSE_VOUCHER_DELETED",
			fmt.Sprintf("Deleted Expense Voucher #%d", id),
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/expenses", http.StatusSeeOther)
}

func (app *Application) adminExportInventoryHandler(w http.ResponseWriter, r *http.Request) {
	if app.InventoryService == nil {
		app.serverError(w, r, fmt.Errorf("inventory service not initialized"))
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "fy"
	}

	csvData, filename, err := app.InventoryService.GenerateInventoryCSV(r.Context(), period)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(csvData)))
	w.WriteHeader(http.StatusOK)
	w.Write(csvData)
}
