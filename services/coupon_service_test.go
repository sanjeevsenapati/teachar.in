package services_test

import (
	"context"
	"testing"
	"time"

	"teachar.in/models"
	"teachar.in/repository"
	"teachar.in/services"
)

func TestCouponService(t *testing.T) {
	tempDir := t.TempDir()
	repo, err := repository.NewMultiFileRepository(tempDir)
	if err != nil {
		t.Fatalf("failed initializing repository: %v", err)
	}

	couponSvc := services.NewCouponService(repo)
	ctx := context.Background()

	// 1. Create Flat Coupon
	c1, err := couponSvc.CreateCoupon(ctx, models.Coupon{
		Code:           "welcome50",
		DiscountType:   "flat",
		DiscountValue:  50,
		MinOrderAmount: 100,
		ExpiryDate:     time.Now().Add(24 * time.Hour),
	}, "Superadmin")
	if err != nil {
		t.Fatalf("failed creating coupon: %v", err)
	}

	if c1.Code != "WELCOME50" {
		t.Fatalf("expected uppercase coupon code WELCOME50, got %s", c1.Code)
	}

	// 2. Validate Coupon with insufficient subtotal
	_, _, _, err = couponSvc.ValidateCoupon(ctx, "WELCOME50", 80)
	if err == nil {
		t.Fatalf("expected error for subtotal below min_order_amount, got nil")
	}

	// 3. Validate Coupon with valid subtotal (subtotal = 200, discount = 50, discounted = 150, tax = 7.5, total = 157.5)
	c, disc, finalTotal, err := couponSvc.ValidateCoupon(ctx, "WELCOME50", 200)
	if err != nil {
		t.Fatalf("unexpected error validating coupon: %v", err)
	}

	if c.Code != "WELCOME50" || disc != 50 || finalTotal != 157.5 {
		t.Fatalf("expected disc=50, finalTotal=157.5, got disc=%.2f, finalTotal=%.2f", disc, finalTotal)
	}

	// 4. Test Single-Use Enforcement
	err = couponSvc.MarkCouponUsed(ctx, "WELCOME50", 101)
	if err != nil {
		t.Fatalf("failed marking coupon used: %v", err)
	}

	// Second validation attempt must fail
	_, _, _, err = couponSvc.ValidateCoupon(ctx, "WELCOME50", 200)
	if err == nil {
		t.Fatalf("expected error validating already-used coupon, got nil")
	}

	// 5. Test Expired Coupon
	_, err = couponSvc.CreateCoupon(ctx, models.Coupon{
		Code:          "EXPIRED10",
		DiscountType:  "percentage",
		DiscountValue: 10,
		ExpiryDate:    time.Now().Add(-1 * time.Hour),
	}, "Superadmin")
	if err == nil {
		t.Fatalf("expected error creating coupon with past expiry date, got nil")
	}
}
