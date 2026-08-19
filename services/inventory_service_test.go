package services_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"teachar.in/models"
	"teachar.in/repository"
	"teachar.in/services"
)

func TestInventoryService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := repository.NewSQLiteRepository(dbPath, tempDir)
	if err != nil {
		t.Fatalf("failed initializing repo: %v", err)
	}
	defer repo.Close()

	invSvc := services.NewInventoryService(repo, repo)
	ctx := context.Background()

	// 1. Test Saving Raw Material Inventory Item
	teaItem := models.InventoryItem{
		Category:      "Raw Material",
		ItemName:      "Assam CTC Tea Leaves",
		Unit:          "kg",
		StockQuantity: 50.0,
		ReorderLevel:  10.0,
		UnitCost:      350.0,
		Supplier:      "Chai Traders Pvt Ltd",
	}

	savedItem, err := invSvc.SaveInventoryItem(ctx, teaItem)
	if err != nil {
		t.Fatalf("failed saving inventory item: %v", err)
	}
	if savedItem.ID == 0 {
		t.Errorf("expected non-zero ID for saved inventory item")
	}
	if savedItem.TotalValue != 17500.0 {
		t.Errorf("expected total value 17500.0, got %f", savedItem.TotalValue)
	}

	// 2. Test Saving Capital Equipment Asset
	espressoItem := models.InventoryItem{
		Category:     "Equipment",
		ItemName:     "Commercial Espresso Machine",
		Unit:         "units",
		StockQuantity: 1.0,
		UnitCost:     125000.0,
		Supplier:     "Cafe Tech India",
		SerialNumber: "EXP-2026-9912",
	}
	savedEquipment, err := invSvc.SaveInventoryItem(ctx, espressoItem)
	if err != nil {
		t.Fatalf("failed saving equipment asset: %v", err)
	}
	if savedEquipment.Status != "Active Asset" {
		t.Errorf("expected 'Active Asset' status, got '%s'", savedEquipment.Status)
	}

	// 3. Test Saving Operating Expense Vouchers
	rentExpense := models.ExpenseEntry{
		Category:       "Rent",
		Title:          "Monthly Cafe Lease",
		SupplierVendor: "MG Road Realty",
		InvoiceNo:      "RENT-AUG-2026",
		TotalAmount:    35000.0,
		PaymentMethod:  "Bank Transfer",
		ExpenseDate:    time.Now(),
	}
	savedRent, err := invSvc.SaveExpense(ctx, rentExpense)
	if err != nil {
		t.Fatalf("failed saving expense entry: %v", err)
	}
	if savedRent.ID == 0 {
		t.Errorf("expected non-zero ID for saved expense entry")
	}

	// 4. Test Generating Tax & Audit Report
	report, err := invSvc.GenerateTaxAuditReport(ctx, "fy")
	if err != nil {
		t.Fatalf("failed generating tax audit report: %v", err)
	}
	if report.OperatingExpenses != 35000.0 {
		t.Errorf("expected OpEx 35000.0, got %f", report.OperatingExpenses)
	}
	if report.TotalCapitalAssets != 125000.0 {
		t.Errorf("expected Capital Assets 125000.0, got %f", report.TotalCapitalAssets)
	}

	// 5. Test Exporting Tax Audit CSV
	csvBytes, filename, err := invSvc.GenerateInventoryCSV(ctx, "fy")
	if err != nil {
		t.Fatalf("failed generating inventory CSV: %v", err)
	}
	if len(csvBytes) == 0 {
		t.Errorf("expected non-empty CSV output")
	}
	if filename == "" {
		t.Errorf("expected non-empty CSV filename")
	}
}
