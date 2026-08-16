package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"teachar.in/models"
	"teachar.in/repository"
)

type CouponService struct {
	couponRepo repository.CouponRepository
}

func NewCouponService(couponRepo repository.CouponRepository) *CouponService {
	return &CouponService{couponRepo: couponRepo}
}

func (s *CouponService) CreateCoupon(ctx context.Context, coupon models.Coupon, actorName string) (*models.Coupon, error) {
	coupon.Code = strings.ToUpper(strings.TrimSpace(coupon.Code))
	if coupon.Code == "" {
		return nil, errors.New("coupon code is required")
	}

	if coupon.DiscountValue <= 0 {
		return nil, errors.New("discount value must be greater than zero")
	}

	if coupon.DiscountType != "flat" && coupon.DiscountType != "percentage" {
		return nil, errors.New("invalid discount type, must be 'flat' or 'percentage'")
	}

	if coupon.DiscountType == "percentage" && coupon.DiscountValue > 100 {
		return nil, errors.New("percentage discount cannot exceed 100%")
	}

	if coupon.ExpiryDate.IsZero() {
		return nil, errors.New("expiry date is required")
	}

	if time.Now().After(coupon.ExpiryDate) {
		return nil, errors.New("expiry date must be in the future")
	}

	coupon.CreatedBy = actorName
	return s.couponRepo.CreateCoupon(ctx, coupon)
}

func (s *CouponService) GetCouponByCode(ctx context.Context, code string) (*models.Coupon, error) {
	return s.couponRepo.GetCouponByCode(ctx, code)
}

func (s *CouponService) GetAllCoupons(ctx context.Context) ([]models.Coupon, error) {
	return s.couponRepo.GetAllCoupons(ctx)
}

// ValidateCoupon checks single-use, expiry, and min subtotal requirement, returning discount amount and final total.
func (s *CouponService) ValidateCoupon(ctx context.Context, code string, subtotal float64) (*models.Coupon, float64, float64, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, 0, subtotal, errors.New("coupon code cannot be empty")
	}

	coupon, err := s.couponRepo.GetCouponByCode(ctx, code)
	if err != nil {
		return nil, 0, subtotal, fmt.Errorf("invalid or non-existent coupon code: %s", code)
	}

	if coupon.IsUsed {
		return nil, 0, subtotal, fmt.Errorf("coupon '%s' has already been used", code)
	}

	if time.Now().After(coupon.ExpiryDate) {
		return nil, 0, subtotal, fmt.Errorf("coupon '%s' expired on %s", code, coupon.ExpiryDate.Format("02 Jan 2006 15:04"))
	}

	if coupon.MinOrderAmount > 0 && subtotal < coupon.MinOrderAmount {
		return nil, 0, subtotal, fmt.Errorf("minimum order amount of ₹%.2f required to use coupon '%s'", coupon.MinOrderAmount, code)
	}

	var discountAmount float64
	if coupon.DiscountType == "flat" {
		discountAmount = coupon.DiscountValue
	} else if coupon.DiscountType == "percentage" {
		discountAmount = subtotal * (coupon.DiscountValue / 100.0)
	}

	if discountAmount > subtotal {
		discountAmount = subtotal
	}

	discountAmount = math.Round(discountAmount*100) / 100
	discountedSubtotal := subtotal - discountAmount
	tax := discountedSubtotal * 0.05
	finalTotal := math.Round((discountedSubtotal+tax)*100) / 100

	return coupon, discountAmount, finalTotal, nil
}

func (s *CouponService) MarkCouponUsed(ctx context.Context, code string, orderID int64) error {
	return s.couponRepo.MarkCouponUsed(ctx, code, orderID)
}

func (s *CouponService) DeleteCoupon(ctx context.Context, id int64) error {
	return s.couponRepo.DeleteCoupon(ctx, id)
}
