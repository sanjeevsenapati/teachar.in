package services_test

import (
	"context"
	"path/filepath"
	"testing"

	"teachar.in/repository"
	"teachar.in/services"
)

func TestMenuService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_db.json")

	repo, err := repository.NewJSONRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	service := services.NewMenuService(repo)
	ctx := context.Background()

	t.Run("GetFullMenu", func(t *testing.T) {
		fullMenu, err := service.GetFullMenu(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fullMenu) == 0 {
			t.Error("expected non-empty menu")
		}
	})

	t.Run("GetFeaturedItems", func(t *testing.T) {
		featured, err := service.GetFeaturedItems(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(featured) == 0 {
			t.Error("expected non-empty featured items")
		}
	})

	t.Run("GetItemByID", func(t *testing.T) {
		item, err := service.GetItemByID(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.ID != 1 {
			t.Errorf("expected item ID 1, got %d", item.ID)
		}
	})
}
