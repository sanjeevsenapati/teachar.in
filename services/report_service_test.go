package services_test

import (
	"context"
	"path/filepath"
	"testing"

	"teachar.in/models"
	"teachar.in/repository"
	"teachar.in/services"
)

func TestReportService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := repository.NewSQLiteRepository(dbPath, tempDir)
	if err != nil {
		t.Fatalf("failed initializing repo: %v", err)
	}
	defer repo.Close()

	reportSvc := services.NewReportService(repo, repo, repo)
	ctx := context.Background()

	// Seed dummy orders
	_, err = repo.CreateOrder(ctx, models.Order{
		OrderType:     "Dine-in",
		Status:        "Completed",
		CustomerName:  "Test Client",
		TableNumber:   "Table 1",
		PaymentStatus: "Paid",
		PaymentMethod: "UPI",
		Items: []models.OrderItem{
			{MenuItemID: 1, ItemName: "Masala Chai", Quantity: 2, Price: 30},
		},
	})
	if err != nil {
		t.Fatalf("failed creating order: %v", err)
	}

	periods := []string{"today", "yesterday", "daily", "weekly", "monthly", "quarterly", "yearly"}
	for _, p := range periods {
		report, err := reportSvc.GenerateFinancialReport(ctx, p)
		if err != nil {
			t.Fatalf("failed generating financial report for period '%s': %v", p, err)
		}

		if report.Period != p {
			t.Errorf("expected period '%s', got '%s'", p, report.Period)
		}

		if len(report.HourlyAnalytics) != 24 {
			t.Errorf("expected 24 hourly points in time series, got %d", len(report.HourlyAnalytics))
		}
	}

	// Test Dynamic Multi-Dimensional Filtering
	filter := models.ReportFilter{
		Period:            "today",
		PaymentMethod:     "UPI",
		FulfillmentMethod: "Dine-in",
		OrderStatus:       "Completed",
	}

	filteredReport, err := reportSvc.GenerateFilteredFinancialReport(ctx, filter)
	if err != nil {
		t.Fatalf("failed generating filtered report: %v", err)
	}
	if filteredReport.TotalOrders != 1 {
		t.Errorf("expected 1 filtered order, got %d", filteredReport.TotalOrders)
	}
}
