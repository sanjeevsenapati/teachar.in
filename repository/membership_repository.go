package repository

import (
	"context"

	"teachar.in/models"
)

// MembershipRepository handles data access for user subscriptions and membership tiers.
type MembershipRepository interface {
	GetAllSubscriptions(ctx context.Context) ([]models.UserSubscription, error)
	GetSubscriptionByUserID(ctx context.Context, userID int64) (*models.UserSubscription, error)
	GetSubscriptionByID(ctx context.Context, id int64) (*models.UserSubscription, error)
	SaveSubscription(ctx context.Context, sub models.UserSubscription) (*models.UserSubscription, error)
	ClaimDailyCup(ctx context.Context, subscriptionID int64) error
	CancelSubscription(ctx context.Context, subscriptionID int64) error
}
