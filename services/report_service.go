package services

import (
	"context"
	"sort"

	"teachar.in/models"
	"teachar.in/repository"
)

type ReportService struct {
	orderRepo repository.OrderRepository
	menuRepo  repository.MenuRepository
}

func NewReportService(orderRepo repository.OrderRepository, menuRepo repository.MenuRepository) *ReportService {
	return &ReportService{
		orderRepo: orderRepo,
		menuRepo:  menuRepo,
	}
}

func (s *ReportService) GenerateFinancialReport(ctx context.Context) (*models.FinancialReportData, error) {
	orders, err := s.orderRepo.GetAllOrders(ctx)
	if err != nil {
		return nil, err
	}

	menuMap, _ := s.menuRepo.GetAll(ctx)
	itemCategoryMap := make(map[string]string)
	for cat, items := range menuMap {
		for _, item := range items {
			itemCategoryMap[item.Name] = cat
		}
	}

	report := &models.FinancialReportData{
		TotalOrders: len(orders),
	}

	paymentMap := map[string]*models.PaymentMethodReport{
		"UPI":        {Method: "UPI"},
		"Card":       {Method: "Card"},
		"NetBanking": {Method: "NetBanking"},
		"COD":        {Method: "COD"},
	}

	orderTypeMap := map[string]*models.OrderTypeReport{
		"Dine-in":  {Type: "Dine-in"},
		"Takeaway": {Type: "Takeaway"},
		"Delivery": {Type: "Delivery"},
	}

	categoryMap := make(map[string]*models.CategoryReport)
	topItemsMap := make(map[string]*models.TopItemReport)

	for _, o := range orders {
		switch o.Status {
		case "Completed":
			report.CompletedOrders++
		case "Pending", "Preparing", "Ready":
			report.PendingOrders++
		case "Cancelled":
			report.CancelledOrders++
		}

		if o.Status == "Cancelled" {
			continue
		}

		report.GrossRevenue += o.TotalPrice
		
		// Subtotal vs 5% GST breakdown
		subtotal := o.TotalPrice / 1.05
		tax := o.TotalPrice - subtotal
		report.TotalTax += tax

		if o.PaymentStatus == "Paid" {
			report.PaidAmount += o.TotalPrice
		} else {
			report.PendingPayment += o.TotalPrice
		}

		// Payment method tally
		if pm, exists := paymentMap[o.PaymentMethod]; exists {
			pm.OrderCount++
			pm.TotalAmount += o.TotalPrice
		}

		// Order type tally
		otKey := o.OrderType
		if otKey == "" {
			otKey = "Dine-in"
		}
		if ot, exists := orderTypeMap[otKey]; exists {
			ot.OrderCount++
			ot.TotalAmount += o.TotalPrice
		}

		// Item & Category sales breakdown
		for _, item := range o.Items {
			cat := itemCategoryMap[item.ItemName]
			if cat == "" {
				cat = "General"
			}

			if _, exists := categoryMap[cat]; !exists {
				categoryMap[cat] = &models.CategoryReport{Category: cat}
			}
			categoryMap[cat].ItemsSold += item.Quantity
			categoryMap[cat].TotalAmount += item.Price * float64(item.Quantity)

			if _, exists := topItemsMap[item.ItemName]; !exists {
				topItemsMap[item.ItemName] = &models.TopItemReport{
					ItemName: item.ItemName,
					Category: cat,
				}
			}
			topItemsMap[item.ItemName].Quantity += item.Quantity
			topItemsMap[item.ItemName].TotalRevenue += item.Price * float64(item.Quantity)
		}
	}

	report.NetRevenue = report.GrossRevenue - report.TotalTax

	nonCancelledOrders := report.TotalOrders - report.CancelledOrders
	if nonCancelledOrders > 0 {
		report.AverageOrderValue = report.GrossRevenue / float64(nonCancelledOrders)
	}

	for _, pm := range paymentMap {
		report.PaymentMethods = append(report.PaymentMethods, *pm)
	}

	for _, ot := range orderTypeMap {
		report.OrderTypes = append(report.OrderTypes, *ot)
	}

	for _, cat := range categoryMap {
		report.CategorySales = append(report.CategorySales, *cat)
	}

	for _, top := range topItemsMap {
		report.TopSellingItems = append(report.TopSellingItems, *top)
	}

	// Sort TopSellingItems by revenue descending
	sort.Slice(report.TopSellingItems, func(i, j int) bool {
		return report.TopSellingItems[i].TotalRevenue > report.TopSellingItems[j].TotalRevenue
	})

	return report, nil
}
