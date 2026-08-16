package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"teachar.in/models"
	"teachar.in/repository"
)

type ReportService struct {
	orderRepo repository.OrderRepository
	menuRepo  repository.MenuRepository
	userRepo  repository.UserRepository
}

func NewReportService(orderRepo repository.OrderRepository, menuRepo repository.MenuRepository, userRepo repository.UserRepository) *ReportService {
	return &ReportService{
		orderRepo: orderRepo,
		menuRepo:  menuRepo,
		userRepo:  userRepo,
	}
}

// GenerateFinancialReport maintains backward compatibility defaulting to period string.
func (s *ReportService) GenerateFinancialReport(ctx context.Context, period string) (*models.FinancialReportData, error) {
	return s.GenerateFilteredFinancialReport(ctx, models.ReportFilter{Period: period})
}

// GenerateFilteredFinancialReport dynamically filters orders by user-specified date ranges, order statuses, payment methods, fulfillment types, categories, and keywords.
func (s *ReportService) GenerateFilteredFinancialReport(ctx context.Context, filter models.ReportFilter) (*models.FinancialReportData, error) {
	allOrders, err := s.orderRepo.GetAllOrders(ctx)
	if err != nil {
		return nil, err
	}

	period := strings.ToLower(strings.TrimSpace(filter.Period))
	if period == "" {
		period = "today"
	}
	filter.Period = period

	now := time.Now()
	var startDate, endDate time.Time
	var periodLabel string

	// Handle Custom Date Range or Preset Periods
	if filter.StartDateStr != "" && filter.EndDateStr != "" {
		sTime, sErr := time.Parse("2006-01-02", filter.StartDateStr)
		eTime, eErr := time.Parse("2006-01-02", filter.EndDateStr)
		if sErr == nil && eErr == nil {
			period = "custom"
			filter.Period = "custom"
			startDate = time.Date(sTime.Year(), sTime.Month(), sTime.Day(), 0, 0, 0, 0, now.Location())
			endDate = time.Date(eTime.Year(), eTime.Month(), eTime.Day(), 23, 59, 59, 999999999, now.Location())
			periodLabel = fmt.Sprintf("Custom Range (%s to %s)", sTime.Format("Jan 02, 2006"), eTime.Format("Jan 02, 2006"))
		}
	}

	if startDate.IsZero() || endDate.IsZero() {
		switch period {
		case "today":
			startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
			periodLabel = "Today Live Data (Default)"

		case "yesterday":
			yesterday := now.AddDate(0, 0, -1)
			startDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
			endDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, now.Location())
			periodLabel = "Yesterday (" + yesterday.Format("Jan 02, 2006") + ")"

		case "daily":
			startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -6)
			endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
			periodLabel = "Past 7 Days (Daily View)"

		case "weekly":
			weekday := int(now.Weekday())
			if weekday == 0 {
				weekday = 7 // Sunday
			}
			startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -weekday+1)
			endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
			periodLabel = "Current Week Overview"

		case "monthly":
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			nextMonth := startDate.AddDate(0, 1, 0)
			endDate = nextMonth.Add(-1 * time.Nanosecond)
			periodLabel = fmt.Sprintf("Current Month (%s %d)", now.Month().String(), now.Year())

		case "quarterly":
			quarterMonth := ((int(now.Month()) - 1) / 3) * 3 + 1
			startDate = time.Date(now.Year(), time.Month(quarterMonth), 1, 0, 0, 0, 0, now.Location())
			endDate = startDate.AddDate(0, 3, 0).Add(-1 * time.Nanosecond)
			periodLabel = fmt.Sprintf("Q%d %d Overview", ((int(now.Month())-1)/3)+1, now.Year())

		case "yearly":
			fyYear := now.Year()
			if now.Month() < time.April {
				fyYear--
			}
			startDate = time.Date(fyYear, time.April, 1, 0, 0, 0, 0, now.Location())
			endDate = time.Date(fyYear+1, time.March, 31, 23, 59, 59, 999999999, now.Location())
			periodLabel = fmt.Sprintf("Financial Year (FY %d-%02d)", fyYear, (fyYear+1)%100)

		default:
			period = "today"
			filter.Period = "today"
			startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
			periodLabel = "Today Live Data (Default)"
		}
	}

	filter.StartDateStr = startDate.Format("2006-01-02")
	filter.EndDateStr = endDate.Format("2006-01-02")

	menuMap, _ := s.menuRepo.GetAll(ctx)
	itemCategoryMap := make(map[string]string)
	for cat, items := range menuMap {
		for _, item := range items {
			itemCategoryMap[item.Name] = cat
		}
	}

	// Dynamic Multi-Dimensional Order Filtering
	var orders []models.Order
	for _, o := range allOrders {
		// 1. Date Range Filter
		if o.CreatedAt.Before(startDate) || o.CreatedAt.After(endDate) {
			continue
		}

		// 2. Order Status Filter
		if filter.OrderStatus != "" && !strings.EqualFold(filter.OrderStatus, "all") {
			if !strings.EqualFold(o.Status, filter.OrderStatus) {
				continue
			}
		}

		// 3. Fulfillment Method Filter
		if filter.FulfillmentMethod != "" && !strings.EqualFold(filter.FulfillmentMethod, "all") {
			if !strings.EqualFold(o.OrderType, filter.FulfillmentMethod) {
				continue
			}
		}

		// 4. Payment Method Filter
		if filter.PaymentMethod != "" && !strings.EqualFold(filter.PaymentMethod, "all") {
			if !strings.EqualFold(o.PaymentMethod, filter.PaymentMethod) {
				continue
			}
		}

		// 5. Category Filter
		if filter.Category != "" && !strings.EqualFold(filter.Category, "all") {
			hasCategoryItem := false
			for _, item := range o.Items {
				cat := itemCategoryMap[item.ItemName]
				if strings.EqualFold(cat, filter.Category) {
					hasCategoryItem = true
					break
				}
			}
			if !hasCategoryItem {
				continue
			}
		}

		// 6. Search Query Keyword Filter
		if filter.SearchQuery != "" {
			q := strings.ToLower(filter.SearchQuery)
			matches := strings.Contains(strings.ToLower(o.CustomerName), q) ||
				strings.Contains(strings.ToLower(o.CustomerPhone), q) ||
				strings.Contains(strings.ToLower(o.TableNumber), q) ||
				strings.Contains(fmt.Sprintf("%d", o.ID), q)

			if !matches {
				for _, item := range o.Items {
					if strings.Contains(strings.ToLower(item.ItemName), q) {
						matches = true
						break
					}
				}
			}
			if !matches {
				continue
			}
		}

		orders = append(orders, o)
	}

	report := &models.FinancialReportData{
		Period:      period,
		PeriodLabel: periodLabel,
		Filter:      filter,
		StartDate:   startDate,
		EndDate:     endDate,
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

	// Initialize 24-Hour Time Series Array (Hours 0 to 23)
	hourlyPoints := make([]models.HourlyPoint, 24)
	for h := 0; h < 24; h++ {
		tVal := time.Date(2026, 1, 1, h, 0, 0, 0, time.UTC)
		hourlyPoints[h] = models.HourlyPoint{
			Hour:  h,
			Label: tVal.Format("03:00 PM"),
		}
	}

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

		// Aggregate 24-hour time series
		h := o.CreatedAt.Hour()
		if h >= 0 && h < 24 {
			hourlyPoints[h].OrderCount++
			hourlyPoints[h].Revenue += o.TotalPrice
		}

		report.GrossRevenue += o.TotalPrice

		// Subtotal vs 5% GST tax breakdown
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

		// Fulfillment method tally
		if ot, exists := orderTypeMap[o.OrderType]; exists {
			ot.OrderCount++
			ot.TotalAmount += o.TotalPrice
		}

		// Item & Category sales breakdown
		for _, item := range o.Items {
			sub := float64(item.Quantity) * item.Price

			cat := itemCategoryMap[item.ItemName]
			if cat == "" {
				cat = "General Snacks"
			}

			// Category tally
			cr, exists := categoryMap[cat]
			if !exists {
				cr = &models.CategoryReport{Category: cat}
				categoryMap[cat] = cr
			}
			cr.ItemsSold += item.Quantity
			cr.TotalAmount += sub

			// Item tally
			ti, tiExists := topItemsMap[item.ItemName]
			if !tiExists {
				ti = &models.TopItemReport{ItemName: item.ItemName, Category: cat}
				topItemsMap[item.ItemName] = ti
			}
			ti.Quantity += item.Quantity
			ti.TotalRevenue += sub
		}
	}

	report.NetRevenue = report.GrossRevenue - report.TotalTax
	validOrderCount := report.CompletedOrders + report.PendingOrders
	if validOrderCount > 0 {
		report.AverageOrderValue = math.Round((report.GrossRevenue/float64(validOrderCount))*100) / 100
	}

	// Determine Peak Rush Hour & Slowest Operating Hour
	var peakHour, slowHour int
	var maxRev float64
	minRev := math.MaxFloat64
	hasPeak := false

	for h, hp := range hourlyPoints {
		if hp.Revenue > maxRev {
			maxRev = hp.Revenue
			peakHour = h
			hasPeak = true
		}
		if hp.Revenue > 0 && hp.Revenue < minRev {
			minRev = hp.Revenue
			slowHour = h
		}
	}

	if hasPeak && maxRev > 0 {
		hourlyPoints[peakHour].IsPeak = true
		startT := time.Date(2026, 1, 1, peakHour, 0, 0, 0, time.UTC).Format("03:00 PM")
		endT := time.Date(2026, 1, 1, (peakHour+1)%24, 0, 0, 0, time.UTC).Format("03:00 PM")
		report.PeakRushHour = fmt.Sprintf("%s - %s (%d Orders, ₹%.2f)", startT, endT, hourlyPoints[peakHour].OrderCount, hourlyPoints[peakHour].Revenue)
	} else {
		report.PeakRushHour = "No peak activity recorded for this filter."
	}

	if minRev < math.MaxFloat64 {
		hourlyPoints[slowHour].IsSlow = true
		startT := time.Date(2026, 1, 1, slowHour, 0, 0, 0, time.UTC).Format("03:00 PM")
		endT := time.Date(2026, 1, 1, (slowHour+1)%24, 0, 0, 0, time.UTC).Format("03:00 PM")
		report.SlowestHour = fmt.Sprintf("%s - %s (%d Orders, ₹%.2f)", startT, endT, hourlyPoints[slowHour].OrderCount, hourlyPoints[slowHour].Revenue)
	} else {
		report.SlowestHour = "No sales recorded during selected period."
	}

	// Data-Driven Dynamic Marketing Recommendation
	if hasPeak && maxRev > 0 {
		startT := time.Date(2026, 1, 1, peakHour, 0, 0, 0, time.UTC).Format("03:00 PM")
		report.MarketingAdvice = fmt.Sprintf("Peak rush detected at %s. Ensure +2 extra staff are scheduled during this window to maintain SLA fulfillment under 15 minutes.", startT)
	} else {
		report.MarketingAdvice = "Promote Happy Hour combos (Tea + Snacks) to boost sales during current period."
	}

	report.HourlyAnalytics = hourlyPoints

	// Flatten Payment Methods
	for _, pm := range paymentMap {
		report.PaymentMethods = append(report.PaymentMethods, *pm)
	}

	// Flatten Order Types
	for _, ot := range orderTypeMap {
		report.OrderTypes = append(report.OrderTypes, *ot)
	}

	// Flatten & Sort Category Sales
	for _, cr := range categoryMap {
		report.CategorySales = append(report.CategorySales, *cr)
	}
	sort.Slice(report.CategorySales, func(i, j int) bool {
		return report.CategorySales[i].TotalAmount > report.CategorySales[j].TotalAmount
	})

	// Flatten & Sort Top Selling Items
	for _, ti := range topItemsMap {
		report.TopSellingItems = append(report.TopSellingItems, *ti)
	}
	sort.Slice(report.TopSellingItems, func(i, j int) bool {
		return report.TopSellingItems[i].Quantity > report.TopSellingItems[j].Quantity
	})

	return report, nil
}

func (s *ReportService) GetStaffPerformanceReport(ctx context.Context) ([]models.StaffPerformanceReport, error) {
	orders, err := s.orderRepo.GetAllOrders(ctx)
	if err != nil {
		return nil, err
	}

	var users []models.User
	if s.userRepo != nil {
		users, _ = s.userRepo.GetAllUsers(ctx)
	}

	staffMap := make(map[int64]*models.StaffPerformanceReport)

	// Pre-populate with all staff members
	for _, u := range users {
		if u.Role == "staff" {
			staffMap[u.ID] = &models.StaffPerformanceReport{
				StaffID:    u.ID,
				StaffName:  u.Name,
				StaffEmail: u.Email,
			}
		}
	}

	// Calculate metrics for each order
	for _, o := range orders {
		if o.AssignedStaffID == 0 {
			continue
		}

		rep, exists := staffMap[o.AssignedStaffID]
		if !exists {
			rep = &models.StaffPerformanceReport{
				StaffID:   o.AssignedStaffID,
				StaffName: o.AssignedStaffName,
			}
			staffMap[o.AssignedStaffID] = rep
		}

		rep.TotalAssignedOrders++

		if o.Status == "Completed" {
			rep.CompletedOrders++

			// 20-minute threshold logic
			duration := o.FulfillmentMinutes
			if duration <= 0 && o.CompletedAt != nil {
				duration = int(o.CompletedAt.Sub(o.CreatedAt).Minutes())
			}
			if duration <= 0 {
				duration = int(time.Since(o.CreatedAt).Minutes())
				if duration < 1 {
					duration = 1
				}
			}

			targetMinutes := o.EstimatedMinutes
			if targetMinutes <= 0 {
				targetMinutes = 20
			}

			if duration <= targetMinutes {
				rep.OnTimeOrders++
			} else {
				rep.OverdueOrders++
			}

			rep.AvgFulfillmentMinutes += float64(duration)

			if o.Rating >= 1 && o.Rating <= 5 {
				rep.TotalRatings++
				rep.AvgRating += float64(o.Rating)
			}
		}
	}

	var reports []models.StaffPerformanceReport
	for _, rep := range staffMap {
		if rep.CompletedOrders > 0 {
			rep.OnTimeRate = math.Round((float64(rep.OnTimeOrders)/float64(rep.CompletedOrders))*1000) / 10
			rep.AvgFulfillmentMinutes = math.Round((rep.AvgFulfillmentMinutes/float64(rep.CompletedOrders))*10) / 10
		}
		if rep.TotalRatings > 0 {
			rep.AvgRating = math.Round((rep.AvgRating/float64(rep.TotalRatings))*10) / 10
		}
		reports = append(reports, *rep)
	}

	// Sort by CompletedOrders descending, then OnTimeRate descending
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].CompletedOrders == reports[j].CompletedOrders {
			return reports[i].OnTimeRate > reports[j].OnTimeRate
		}
		return reports[i].CompletedOrders > reports[j].CompletedOrders
	})

	return reports, nil
}
