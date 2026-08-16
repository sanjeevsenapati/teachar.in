package services_test

import (
	"context"
	"testing"

	"teachar.in/models"
	"teachar.in/repository"
	"teachar.in/services"
)

func TestReportService(t *testing.T) {
	tempDir := t.TempDir()
	repo, err := repository.NewMultiFileRepository(tempDir)
	if err != nil {
		t.Fatalf("failed initializing repo: %v", err)
	}

	reportSvc := services.NewReportService(repo, repo, repo)
	ctx := context.Background()

	// Seed dummy orders
	_, err = repo.CreateOrder(ctx, models.Order{
		OrderType:    "Dine-in",
		CustomerName: "Test Client",
		TableNumber:  "Table 1",
		PaymentStatus: "Paid",
		PaymentMethod: "UPI",
		Items: []models.OrderItem{
			{MenuItemID: 1, Quantity: 2, Price: 30},
		},
	})
	if err != nil {
		t.Fatalf("failed creating order: %v", err)
	}

	periods := []string{"today", "daily", "weekly", "monthly", "yearly"}
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

		if report.TotalOrders == 0 {
			t.Errorf("expected at least 1 total order for period '%s', got 0", p)
		}
	}
}
