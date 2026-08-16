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

func (s *ReportService) GenerateFinancialReport(ctx context.Context, period string) (*models.FinancialReportData, error) {
	allOrders, err := s.orderRepo.GetAllOrders(ctx)
	if err != nil {
		return nil, err
	}

	period = strings.ToLower(strings.TrimSpace(period))
	if period == "" {
		period = "today"
	}

	now := time.Now()
	var startDate, endDate time.Time
	var periodLabel string

	switch period {
	case "today":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
		periodLabel = "Today (Live Data)"

	case "daily":
		// Past 7 days daily breakdown
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -6)
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
		periodLabel = "Past 7 Days (Daily View)"

	case "weekly":
		// Current week
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday
		}
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -weekday+1)
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
		periodLabel = "Current Week Overview"

	case "monthly":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		// Last day of month
		nextMonth := startDate.AddDate(0, 1, 0)
		endDate = nextMonth.Add(-1 * time.Nanosecond)
		periodLabel = fmt.Sprintf("Current Month (%s %d)", now.Month().String(), now.Year())

	case "yearly":
		// Indian Financial Year (April 1 to March 31)
		fyYear := now.Year()
		if now.Month() < time.April {
			fyYear--
		}
		startDate = time.Date(fyYear, time.April, 1, 0, 0, 0, 0, now.Location())
		endDate = time.Date(fyYear+1, time.March, 31, 23, 59, 59, 999999999, now.Location())
		periodLabel = fmt.Sprintf("Financial Year (FY %d-%02d)", fyYear, (fyYear+1)%100)

	default:
		period = "today"
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
		periodLabel = "Today (Live Data)"
	}

	// Filter orders by date range
	var orders []models.Order
	for _, o := range allOrders {
		if (o.CreatedAt.Equal(startDate) || o.CreatedAt.After(startDate)) &&
			(o.CreatedAt.Equal(endDate) || o.CreatedAt.Before(endDate)) {
			orders = append(orders, o)
		}
	}

	menuMap, _ := s.menuRepo.GetAll(ctx)
	itemCategoryMap := make(map[string]string)
	for cat, items := range menuMap {
		for _, item := range items {
			itemCategoryMap[item.Name] = cat
		}
	}

	report := &models.FinancialReportData{
		Period:      period,
		PeriodLabel: periodLabel,
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

	report.NetRevenue = math.Round((report.GrossRevenue-report.TotalTax)*100) / 100
	report.TotalTax = math.Round(report.TotalTax*100) / 100
	report.GrossRevenue = math.Round(report.GrossRevenue*100) / 100

	nonCancelledOrders := report.TotalOrders - report.CancelledOrders
	if nonCancelledOrders > 0 {
		report.AverageOrderValue = math.Round((report.GrossRevenue/float64(nonCancelledOrders))*100) / 100
	}

	// Identify Rush Hour (Peak) and Quiet Hour (Slowest)
	peakHourIdx := -1
	maxOrders := -1
	maxRev := -1.0

	slowHourIdx := -1
	minOrders := math.MaxInt32

	for h := 0; h < 24; h++ {
		// Round revenue
		hourlyPoints[h].Revenue = math.Round(hourlyPoints[h].Revenue*100) / 100

		cnt := hourlyPoints[h].OrderCount
		rev := hourlyPoints[h].Revenue

		if cnt > maxOrders || (cnt == maxOrders && rev > maxRev) {
			maxOrders = cnt
			maxRev = rev
			peakHourIdx = h
		}

		if cnt < minOrders {
			minOrders = cnt
			slowHourIdx = h
		}
	}

	if peakHourIdx >= 0 && maxOrders > 0 {
		hourlyPoints[peakHourIdx].IsPeak = true
		peakNextHour := (peakHourIdx + 1) % 24
		t1 := time.Date(2026, 1, 1, peakHourIdx, 0, 0, 0, time.UTC).Format("03:00 PM")
		t2 := time.Date(2026, 1, 1, peakNextHour, 0, 0, 0, time.UTC).Format("03:00 PM")
		report.PeakRushHour = fmt.Sprintf("%s – %s (%d Orders, ₹%.2f)", t1, t2, maxOrders, maxRev)
	} else {
		report.PeakRushHour = "No peak rush recorded yet"
	}

	if slowHourIdx >= 0 {
		hourlyPoints[slowHourIdx].IsSlow = true
		slowNextHour := (slowHourIdx + 1) % 24
		t1 := time.Date(2026, 1, 1, slowHourIdx, 0, 0, 0, time.UTC).Format("03:00 PM")
		t2 := time.Date(2026, 1, 1, slowNextHour, 0, 0, 0, time.UTC).Format("03:00 PM")
		slowRev := hourlyPoints[slowHourIdx].Revenue
		report.SlowestHour = fmt.Sprintf("%s – %s (%d Orders, ₹%.2f)", t1, t2, hourlyPoints[slowHourIdx].OrderCount, slowRev)
	} else {
		report.SlowestHour = "N/A"
	}

	report.HourlyAnalytics = hourlyPoints

	// Automated Marketing Advice
	if peakHourIdx >= 0 && maxOrders > 0 {
		peakTimeStr := time.Date(2026, 1, 1, peakHourIdx, 0, 0, 0, time.UTC).Format("03:00 PM")
		slowTimeStr := time.Date(2026, 1, 1, slowHourIdx, 0, 0, 0, time.UTC).Format("03:00 PM")

		report.MarketingAdvice = fmt.Sprintf(
			"💡 Cafe Marketing Tip: Peak rush occurs around %s (%d orders). Ensure maximum staff coverage and prep fast-moving items in advance. To boost footfall during quieter hours (%s), consider launching a 'Happy Hour' 15%% discount coupon or combo offer.",
			peakTimeStr, maxOrders, slowTimeStr,
		)
	} else {
		report.MarketingAdvice = "💡 Cafe Marketing Tip: Encourage initial orders by launching 'WELCOME50' or 'CHAI10' promo coupons across your social channels!"
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

	sort.Slice(report.TopSellingItems, func(i, j int) bool {
		return report.TopSellingItems[i].TotalRevenue > report.TopSellingItems[j].TotalRevenue
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
