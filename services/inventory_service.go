package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"sort"
	"time"

	"teachar.in/models"
	"teachar.in/repository"
)

type InventoryService struct {
	inventoryRepo repository.InventoryRepository
	orderRepo     repository.OrderRepository
}

func NewInventoryService(invRepo repository.InventoryRepository, orderRepo repository.OrderRepository) *InventoryService {
	return &InventoryService{
		inventoryRepo: invRepo,
		orderRepo:     orderRepo,
	}
}

func (s *InventoryService) GetAllInventory(ctx context.Context) ([]models.InventoryItem, error) {
	items, err := s.inventoryRepo.GetAllInventoryItems(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID > items[j].ID
	})
	return items, nil
}

func (s *InventoryService) SaveInventoryItem(ctx context.Context, item models.InventoryItem) (*models.InventoryItem, error) {
	if item.ItemName == "" {
		return nil, fmt.Errorf("item name is required")
	}
	if item.Category == "" {
		item.Category = "Raw Material"
	}
	if item.Unit == "" {
		item.Unit = "units"
	}
	return s.inventoryRepo.SaveInventoryItem(ctx, item)
}

func (s *InventoryService) DeleteInventoryItem(ctx context.Context, id int64) error {
	return s.inventoryRepo.DeleteInventoryItem(ctx, id)
}

func (s *InventoryService) GetAllExpenses(ctx context.Context) ([]models.ExpenseEntry, error) {
	expenses, err := s.inventoryRepo.GetAllExpenses(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(expenses, func(i, j int) bool {
		return expenses[i].ExpenseDate.After(expenses[j].ExpenseDate)
	})
	return expenses, nil
}

func (s *InventoryService) SaveExpense(ctx context.Context, expense models.ExpenseEntry) (*models.ExpenseEntry, error) {
	if expense.Title == "" {
		return nil, fmt.Errorf("expense title/description is required")
	}
	if expense.Category == "" {
		expense.Category = "Miscellaneous"
	}
	if expense.TotalAmount <= 0 && expense.Quantity > 0 && expense.UnitPrice > 0 {
		expense.TotalAmount = expense.Quantity * expense.UnitPrice
	}
	if expense.TotalAmount <= 0 {
		return nil, fmt.Errorf("expense amount must be greater than zero")
	}
	return s.inventoryRepo.SaveExpense(ctx, expense)
}

func (s *InventoryService) DeleteExpense(ctx context.Context, id int64) error {
	return s.inventoryRepo.DeleteExpense(ctx, id)
}

func (s *InventoryService) GenerateTaxAuditReport(ctx context.Context, period string) (*models.TaxAuditReport, error) {
	orders, err := s.orderRepo.GetAllOrders(ctx)
	if err != nil {
		return nil, err
	}

	inventory, err := s.inventoryRepo.GetAllInventoryItems(ctx)
	if err != nil {
		return nil, err
	}

	expenses, err := s.inventoryRepo.GetAllExpenses(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var startDate time.Time

	switch period {
	case "today":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "daily":
		startDate = now.AddDate(0, 0, -30)
	case "monthly":
		startDate = now.AddDate(0, -1, 0)
	case "yearly", "fy":
		// Standard Financial Year (Apr 1 to Mar 31)
		if now.Month() >= time.April {
			startDate = time.Date(now.Year(), time.April, 1, 0, 0, 0, 0, now.Location())
		} else {
			startDate = time.Date(now.Year()-1, time.April, 1, 0, 0, 0, 0, now.Location())
		}
	default:
		period = "fy"
		if now.Month() >= time.April {
			startDate = time.Date(now.Year(), time.April, 1, 0, 0, 0, 0, now.Location())
		} else {
			startDate = time.Date(now.Year()-1, time.April, 1, 0, 0, 0, 0, now.Location())
		}
	}

	// 1. Calculate Sales Revenue
	var grossRevenue float64
	for _, o := range orders {
		if o.Status == "Completed" && (o.CreatedAt.After(startDate) || o.CreatedAt.Equal(startDate)) {
			grossRevenue += o.TotalPrice
		}
	}

	// 2. Calculate Capital Asset Valuation (Equipment & Furniture)
	var capitalAssets float64
	for _, item := range inventory {
		if item.Category == "Equipment" || item.Category == "Furniture" {
			capitalAssets += item.TotalValue
		}
	}

	// 3. Calculate Expenses Breakdown
	var cogs float64
	var opEx float64
	categoryBreakdown := make(map[string]float64)

	for _, exp := range expenses {
		if exp.ExpenseDate.After(startDate) || exp.ExpenseDate.Equal(startDate) {
			categoryBreakdown[exp.Category] += exp.TotalAmount

			if exp.Category == "Raw Materials" || exp.Category == "Consumables" {
				cogs += exp.TotalAmount
			} else if exp.Category != "Equipment" && exp.Category != "Furniture" {
				opEx += exp.TotalAmount
			}
		}
	}

	totalExpenditure := cogs + opEx
	netProfit := grossRevenue - totalExpenditure
	taxableInc := netProfit
	if taxableInc < 0 {
		taxableInc = 0
	}

	// 5% standard GST on hospitality sales + 25% corporate tax estimation on net profit
	estimatedGST := math.Round((grossRevenue*0.05)*100) / 100
	estimatedTax := math.Round((taxableInc*0.25)*100) / 100

	return &models.TaxAuditReport{
		Period:              period,
		GrossRevenue:        math.Round(grossRevenue*100) / 100,
		COGS:                math.Round(cogs*100) / 100,
		OperatingExpenses:   math.Round(opEx*100) / 100,
		TotalCapitalAssets:  math.Round(capitalAssets*100) / 100,
		TotalExpenditure:    math.Round(totalExpenditure*100) / 100,
		NetOperatingProfit:  math.Round(netProfit*100) / 100,
		EstimatedTaxableInc: math.Round(taxableInc*100) / 100,
		EstimatedGST:        estimatedGST,
		EstimatedIncomeTax:  estimatedTax,
		CategoryBreakdown:   categoryBreakdown,
	}, nil
}

func (s *InventoryService) GenerateInventoryCSV(ctx context.Context, period string) ([]byte, string, error) {
	report, err := s.GenerateTaxAuditReport(ctx, period)
	if err != nil {
		return nil, "", err
	}

	inventory, _ := s.inventoryRepo.GetAllInventoryItems(ctx)
	expenses, _ := s.inventoryRepo.GetAllExpenses(ctx)

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write Audit Summary Section
	writer.Write([]string{"--- TEACHAR.IN CAFE FINANCIAL YEAR & TAX AUDIT REPORT ---"})
	writer.Write([]string{"Report Period", report.Period})
	writer.Write([]string{"Generated Date", time.Now().Format("2006-01-02 15:04:05")})
	writer.Write([]string{})

	writer.Write([]string{"FINANCIAL AUDIT SUMMARY METRICS", "AMOUNT (INR)"})
	writer.Write([]string{"Gross Cafe Sales Revenue", fmt.Sprintf("%.2f", report.GrossRevenue)})
	writer.Write([]string{"Cost of Goods Sold (COGS - Raw Materials)", fmt.Sprintf("%.2f", report.COGS)})
	writer.Write([]string{"Operating Expenses (OpEx - Rent, Utilities, Freight)", fmt.Sprintf("%.2f", report.OperatingExpenses)})
	writer.Write([]string{"Total Capital Equipment & Furniture Asset Value", fmt.Sprintf("%.2f", report.TotalCapitalAssets)})
	writer.Write([]string{"Total Expenditures (COGS + OpEx)", fmt.Sprintf("%.2f", report.TotalExpenditure)})
	writer.Write([]string{"Net Operating EBITDA Profit / Loss", fmt.Sprintf("%.2f", report.NetOperatingProfit)})
	writer.Write([]string{"Estimated Taxable Net Income", fmt.Sprintf("%.2f", report.EstimatedTaxableInc)})
	writer.Write([]string{"Estimated Output GST Payable (5%)", fmt.Sprintf("%.2f", report.EstimatedGST)})
	writer.Write([]string{"Estimated Income Tax (25%)", fmt.Sprintf("%.2f", report.EstimatedIncomeTax)})
	writer.Write([]string{})

	// Write Store Inventory & Assets Section
	writer.Write([]string{"STORE INVENTORY & CAPITAL ASSETS REGISTER"})
	writer.Write([]string{"Item ID", "Category", "Item Name", "Quantity", "Unit", "Unit Cost (INR)", "Total Value (INR)", "Supplier", "Status"})
	for _, item := range inventory {
		writer.Write([]string{
			fmt.Sprintf("%d", item.ID),
			item.Category,
			item.ItemName,
			fmt.Sprintf("%.2f", item.StockQuantity),
			item.Unit,
			fmt.Sprintf("%.2f", item.UnitCost),
			fmt.Sprintf("%.2f", item.TotalValue),
			item.Supplier,
			item.Status,
		})
	}
	writer.Write([]string{})

	// Write Expenditure Vouchers Section
	writer.Write([]string{"EXPENSE VOUCHERS & PURCHASES LOG"})
	writer.Write([]string{"Voucher ID", "Category", "Description", "Supplier/Vendor", "Invoice #", "Amount (INR)", "Payment Method", "Expense Date"})
	for _, exp := range expenses {
		writer.Write([]string{
			fmt.Sprintf("%d", exp.ID),
			exp.Category,
			exp.Title,
			exp.SupplierVendor,
			exp.InvoiceNo,
			fmt.Sprintf("%.2f", exp.TotalAmount),
			exp.PaymentMethod,
			exp.ExpenseDate.Format("2006-01-02"),
		})
	}

	writer.Flush()
	filename := fmt.Sprintf("teachar_store_inventory_tax_audit_%s.csv", report.Period)
	return buf.Bytes(), filename, nil
}
