package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"teachar.in/models"
	"teachar.in/repository"
)

type MembershipService struct {
	repo repository.MembershipRepository
}

func NewMembershipService(repo repository.MembershipRepository) *MembershipService {
	return &MembershipService{repo: repo}
}

// GetMembershipTiers returns available cafe membership pass tiers.
func (s *MembershipService) GetMembershipTiers() []models.MembershipTier {
	return []models.MembershipTier{
		{
			ID:                 "silver",
			Name:               "Silver Chai Pass",
			PriceMonthly:       499.00,
			DiscountPercentage: 10,
			DailyFreeCupLimit:  1,
			EligibleCategories: []string{"Tea"},
			Perks: []string{
				"1 Free Artisanal Chai / Tea Every Day",
				"10% Member Discount on All Snacks",
				"Loyalty Reward Points Tracking",
			},
		},
		{
			ID:                 "gold",
			Name:               "Gold Coffee Pass",
			PriceMonthly:       999.00,
			DiscountPercentage: 15,
			DailyFreeCupLimit:  1,
			EligibleCategories: []string{"Tea", "Coffee"},
			Perks: []string{
				"1 Free Gourmet Coffee / Tea Every Day",
				"15% Member Discount on Entire Menu",
				"Priority Order Fulfillment Tag",
				"Free Birthday Treat Voucher",
			},
		},
		{
			ID:                 "platinum",
			Name:               "Platinum Executive VIP Pass",
			PriceMonthly:       1999.00,
			DiscountPercentage: 20,
			DailyFreeCupLimit:  2,
			EligibleCategories: []string{"Tea", "Coffee", "Cold Beverages"},
			Perks: []string{
				"2 Free Gourmet Beverages Every Day",
				"20% VIP Discount on All Menu Items",
				"Free Home Delivery on All Orders",
				"Exclusive VIP Lounge Access",
				"Dedicated Support Line",
			},
		},
	}
}

func (s *MembershipService) GetTierByID(tierID string) (*models.MembershipTier, error) {
	tiers := s.GetMembershipTiers()
	for _, t := range tiers {
		if t.ID == tierID {
			return &t, nil
		}
	}
	return nil, errors.New("membership tier not found")
}

func (s *MembershipService) SubscribeUser(ctx context.Context, user *models.User, tierID, paymentMethod string) (*models.UserSubscription, error) {
	if user == nil {
		return nil, errors.New("user required")
	}

	tier, err := s.GetTierByID(tierID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	endDate := now.AddDate(0, 1, 0) // 1 Month duration

	txnID := fmt.Sprintf("SUB-%d-%d", now.Unix(), user.ID)

	sub := models.UserSubscription{
		UserID:            user.ID,
		UserName:          user.Name,
		UserEmail:         user.Email,
		TierID:            tier.ID,
		TierName:          tier.Name,
		Status:            "Active",
		PricePaid:         tier.PriceMonthly,
		DiscountPercent:   tier.DiscountPercentage,
		DailyFreeCupLimit: tier.DailyFreeCupLimit,
		CupsClaimedToday:  0,
		StartDate:         now,
		EndDate:           endDate,
		AutoRenew:         true,
		PaymentMethod:     paymentMethod,
		TransactionID:     txnID,
	}

	return s.repo.SaveSubscription(ctx, sub)
}

func (s *MembershipService) GrantUserSubscription(ctx context.Context, user *models.User, tierID string, durationMonths int, grantedBy string) (*models.UserSubscription, error) {
	if user == nil {
		return nil, errors.New("user required")
	}

	tier, err := s.GetTierByID(tierID)
	if err != nil {
		return nil, err
	}

	if durationMonths <= 0 {
		durationMonths = 1
	}

	now := time.Now()
	endDate := now.AddDate(0, durationMonths, 0)

	txnID := fmt.Sprintf("GRANT-%d-%d", now.Unix(), user.ID)

	sub := models.UserSubscription{
		UserID:            user.ID,
		UserName:          user.Name,
		UserEmail:         user.Email,
		TierID:            tier.ID,
		TierName:          tier.Name,
		Status:            "Active",
		PricePaid:         0.0, // Complimentary Admin Grant
		DiscountPercent:   tier.DiscountPercentage,
		DailyFreeCupLimit: tier.DailyFreeCupLimit,
		CupsClaimedToday:  0,
		StartDate:         now,
		EndDate:           endDate,
		AutoRenew:         false,
		PaymentMethod:     fmt.Sprintf("Granted by %s", grantedBy),
		TransactionID:     txnID,
	}

	return s.repo.SaveSubscription(ctx, sub)
}

func (s *MembershipService) GetUserSubscription(ctx context.Context, userID int64) (*models.UserSubscription, error) {
	sub, err := s.repo.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if now.After(sub.EndDate) {
		sub.Status = "Expired"
		_, _ = s.repo.SaveSubscription(ctx, *sub)
		return nil, errors.New("subscription expired")
	}

	// Midnight Reset Check for Daily Cup Claims
	if sub.LastClaimDate == nil || sub.LastClaimDate.Day() != now.Day() || sub.LastClaimDate.Month() != now.Month() {
		sub.CupsClaimedToday = 0
	}

	return sub, nil
}

func (s *MembershipService) ClaimDailyCup(ctx context.Context, userID int64) error {
	sub, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.ClaimDailyCup(ctx, sub.ID)
}

func (s *MembershipService) CalculateMemberDiscount(ctx context.Context, userID int64, subtotal float64) (float64, *models.UserSubscription) {
	sub, err := s.GetUserSubscription(ctx, userID)
	if err != nil || sub == nil || sub.Status != "Active" {
		return 0, nil
	}

	discount := (subtotal * sub.DiscountPercent) / 100.0
	return discount, sub
}

func (s *MembershipService) GetAllSubscriptions(ctx context.Context) ([]models.UserSubscription, error) {
	subs, err := s.repo.GetAllSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(subs, func(i, j int) bool {
		return subs[i].StartDate.After(subs[j].StartDate)
	})
	return subs, nil
}

func (s *MembershipService) CancelSubscription(ctx context.Context, id int64) error {
	return s.repo.CancelSubscription(ctx, id)
}
