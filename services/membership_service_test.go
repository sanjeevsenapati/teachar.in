package services_test

import (
	"context"
	"path/filepath"
	"testing"

	"teachar.in/models"
	"teachar.in/repository"
	"teachar.in/services"
)

func TestMembershipService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := repository.NewSQLiteRepository(dbPath, tempDir)
	if err != nil {
		t.Fatalf("failed initializing repository: %v", err)
	}
	defer repo.Close()

	memSvc := services.NewMembershipService(repo)
	ctx := context.Background()

	// 1. Verify Tiers
	tiers := memSvc.GetMembershipTiers()
	if len(tiers) != 3 {
		t.Fatalf("expected 3 membership tiers, got %d", len(tiers))
	}

	user := &models.User{
		ID:    101,
		Name:  "Test VIP Member",
		Email: "vip@teachar.in",
	}

	// 2. Subscribe User to Gold VIP Membership (₹999/mo, 15% OFF)
	sub, err := memSvc.SubscribeUser(ctx, user, "gold", "UPI")
	if err != nil {
		t.Fatalf("failed subscribing user: %v", err)
	}
	if sub.Status != "Active" {
		t.Errorf("expected active subscription, got %s", sub.Status)
	}
	if sub.DiscountPercent != 15 {
		t.Errorf("expected 15%% discount, got %.0f%%", sub.DiscountPercent)
	}

	// 3. Get User Active Subscription
	activeSub, err := memSvc.GetUserSubscription(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed getting user subscription: %v", err)
	}
	if activeSub.TierID != "gold" {
		t.Errorf("expected tier 'gold', got '%s'", activeSub.TierID)
	}

	// 4. Claim Daily Free Cup
	err = memSvc.ClaimDailyCup(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed claiming daily free cup: %v", err)
	}

	// Re-claim should fail as limit is 1 cup/day
	err = memSvc.ClaimDailyCup(ctx, user.ID)
	if err == nil {
		t.Errorf("expected error claiming second cup when limit is 1 cup/day, got nil")
	}

	// 5. Member Discount Calculation
	subtotal := 500.0 // ₹500 subtotal
	discount, memberSub := memSvc.CalculateMemberDiscount(ctx, user.ID, subtotal)
	if memberSub == nil {
		t.Fatalf("expected non-nil member subscription during discount calculation")
	}
	if discount != 75.0 { // 15% of 500 = 75
		t.Errorf("expected ₹75 discount, got ₹%.2f", discount)
	}

	// 6. Test Subscriber Order Price Adjustment in OrderService.CreateOrder
	couponSvc := services.NewCouponService(repo)
	orderSvc := services.NewOrderService(repo, couponSvc, memSvc)

	orderInput := models.Order{
		UserID:        user.ID,
		CustomerName:  user.Name,
		CustomerPhone: "9876543210",
		OrderType:     "Dine-in",
		TableNumber:   "Table 5",
		PaymentMethod: "UPI",
		Items: []models.OrderItem{
			{MenuItemID: 1, ItemName: "Masala Chai", Price: 100.0, Quantity: 2}, // Subtotal = ₹200
		},
	}

	createdOrder, err := orderSvc.CreateOrder(ctx, orderInput)
	if err != nil {
		t.Fatalf("failed creating order for subscriber: %v", err)
	}

	// 15% subscriber discount on ₹200 = ₹30.00
	if createdOrder.SubscriberDiscount != 30.0 {
		t.Errorf("expected subscriber discount of ₹30.00, got ₹%.2f", createdOrder.SubscriberDiscount)
	}
	if createdOrder.SubscriberTierName != "Gold VIP Membership" {
		t.Errorf("expected tier name 'Gold VIP Membership', got '%s'", createdOrder.SubscriberTierName)
	}
	// Discounted Subtotal = 200 - 30 = 170. 5% GST on 170 = 8.50. Total = 178.50
	if createdOrder.TotalPrice != 178.50 {
		t.Errorf("expected total price of ₹178.50, got ₹%.2f", createdOrder.TotalPrice)
	}
}
