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

// ValidateCoupon checks single-use, expiry, min subtotal, and user targeting.
func (s *CouponService) ValidateCoupon(ctx context.Context, code string, subtotal float64) (*models.Coupon, float64, float64, error) {
	return s.ValidateCouponForUser(ctx, code, subtotal, 0)
}

func (s *CouponService) ValidateCouponForUser(ctx context.Context, code string, subtotal float64, userID int64) (*models.Coupon, float64, float64, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, 0, subtotal, errors.New("coupon code cannot be empty")
	}

	// 1. Check for Virtual Member VIP Coupons (SILVERVIP, GOLDVIP, PLATINUMVIP, VIPPASS)
	if strings.HasSuffix(code, "VIP") || strings.HasSuffix(code, "PASS") || strings.HasPrefix(code, "VIP") {
		var discPercent float64 = 0
		var tierName string = "VIP Membership"
		switch code {
		case "SILVERVIP", "SILVERPASS":
			discPercent = 10
			tierName = "Silver VIP Membership"
		case "GOLDVIP", "GOLDPASS":
			discPercent = 15
			tierName = "Gold VIP Membership"
		case "PLATINUMVIP", "PLATINUMPASS", "VIPPASS", "VIPMEMBERSHIP":
			discPercent = 20
			tierName = "Platinum VIP Membership"
		}

		if discPercent > 0 {
			discountAmount := math.Round(subtotal*(discPercent/100.0)*100) / 100
			discountedSubtotal := subtotal - discountAmount
			tax := discountedSubtotal * 0.05
			finalTotal := math.Round((discountedSubtotal+tax)*100) / 100

			virtualCoupon := &models.Coupon{
				Code:           code,
				DiscountType:   "percentage",
				DiscountValue:  discPercent,
				CreatedBy:      tierName,
				MinOrderAmount: 0,
			}
			return virtualCoupon, discountAmount, finalTotal, nil
		}
	}

	// 2. Standard Stored Coupon Check
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

	if coupon.TargetUserID > 0 && userID > 0 && coupon.TargetUserID != userID {
		return nil, 0, subtotal, fmt.Errorf("coupon '%s' is an exclusive offer for another customer", code)
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

func (s *CouponService) GetAvailableCouponsForUser(ctx context.Context, userID int64) ([]models.Coupon, error) {
	allCoupons, err := s.couponRepo.GetAllCoupons(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var available []models.Coupon
	for _, c := range allCoupons {
		if c.IsUsed {
			continue
		}
		if !c.ExpiryDate.IsZero() && now.After(c.ExpiryDate) {
			continue
		}
		if c.TargetUserID == 0 || c.TargetUserID == userID {
			available = append(available, c)
		}
	}
	return available, nil
}

func (s *CouponService) MarkCouponUsed(ctx context.Context, code string, orderID int64) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if strings.HasSuffix(code, "VIP") || strings.HasSuffix(code, "PASS") || strings.HasPrefix(code, "VIP") {
		// Member coupons are reusable per active subscription pass
		return nil
	}
	return s.couponRepo.MarkCouponUsed(ctx, code, orderID)
}

func (s *CouponService) DeleteCoupon(ctx context.Context, id int64) error {
	return s.couponRepo.DeleteCoupon(ctx, id)
}
