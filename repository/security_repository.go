package repository

import (
	"context"

	"teachar.in/models"
)

// SecurityRepository handles data access for security tokens and API keys.
type SecurityRepository interface {
	GetAllAPIKeys(ctx context.Context) ([]models.APIKey, error)
	GetAPIKeyByID(ctx context.Context, id int64) (*models.APIKey, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*models.APIKey, error)
	SaveAPIKey(ctx context.Context, apiKey models.APIKey) (*models.APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, id int64) error
	RevokeAPIKey(ctx context.Context, id int64) error
}
