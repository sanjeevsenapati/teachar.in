package services

import (
	"context"

	"teachar.in/models"
	"teachar.in/repository"
)

// MenuService provides business logic for menu-related operations.
type MenuService struct {
	repo repository.MenuRepository
}

// NewMenuService creates a new MenuService.
func NewMenuService(repo repository.MenuRepository) *MenuService {
	return &MenuService{repo: repo}
}

// GetFullMenu retrieves the entire menu, organized by category.
func (s *MenuService) GetFullMenu(ctx context.Context) (map[string][]models.MenuItem, error) {
	return s.repo.GetAll(ctx)
}

// GetFeaturedItems retrieves a small selection of items for the home page.
func (s *MenuService) GetFeaturedItems(ctx context.Context) ([]models.MenuItem, error) {
	return s.repo.GetFeatured(ctx)
}

// GetItemByID retrieves a single menu item.
func (s *MenuService) GetItemByID(ctx context.Context, id int64) (*models.MenuItem, error) {
	return s.repo.GetByID(ctx, id)
}

// CreateItem adds a new menu item (Admin).
func (s *MenuService) CreateItem(ctx context.Context, item models.MenuItem) (*models.MenuItem, error) {
	return s.repo.Create(ctx, item)
}

// UpdateItem updates an existing menu item (Admin).
func (s *MenuService) UpdateItem(ctx context.Context, item models.MenuItem) error {
	return s.repo.Update(ctx, item)
}

// DeleteItem removes a menu item (Admin).
func (s *MenuService) DeleteItem(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// ToggleAvailability toggles item availability (Admin).
func (s *MenuService) ToggleAvailability(ctx context.Context, id int64) error {
	return s.repo.ToggleAvailability(ctx, id)
}
