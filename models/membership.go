package models

import (
	"time"
)

// MembershipTier represents a cafe membership tier / subscription plan.
type MembershipTier struct {
	ID                 string   `json:"id"`                  // "silver", "gold", "platinum"
	Name               string   `json:"name"`                // "Gold Coffee Pass"
	PriceMonthly       float64  `json:"price_monthly"`       // ₹999.00
	DiscountPercentage float64  `json:"discount_percentage"` // 15%
	DailyFreeCupLimit  int      `json:"daily_free_cup_limit"`// 1 cup/day
	EligibleCategories []string `json:"eligible_categories"` // ["Tea", "Coffee"]
	Perks              []string `json:"perks"`
}

// UserSubscription represents an active or past user membership subscription pass.
type UserSubscription struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	UserName          string     `json:"user_name"`
	UserEmail         string     `json:"user_email"`
	TierID            string     `json:"tier_id"`
	TierName          string     `json:"tier_name"`
	Status            string     `json:"status"`            // "Active", "Expired", "Cancelled"
	PricePaid         float64    `json:"price_paid"`
	DiscountPercent   float64    `json:"discount_percent"`
	DailyFreeCupLimit int        `json:"daily_free_cup_limit"`
	CupsClaimedToday  int        `json:"cups_claimed_today"`
	LastClaimDate     *time.Time `json:"last_claim_date,omitempty"`
	StartDate         time.Time  `json:"start_date"`
	EndDate           time.Time  `json:"end_date"`
	AutoRenew         bool       `json:"auto_renew"`
	PaymentMethod     string     `json:"payment_method"`
	TransactionID     string     `json:"transaction_id"`
}
