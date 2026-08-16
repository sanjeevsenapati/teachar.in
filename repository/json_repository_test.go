package repository_test

import (
	"context"
	"path/filepath"
	"testing"

	"teachar.in/models"
	"teachar.in/repository"
)

func TestJSONRepository(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_db.json")

	repo, err := repository.NewJSONRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	ctx := context.Background()

	t.Run("Get Menu Items", func(t *testing.T) {
		menuMap, err := repo.GetAll(ctx)
		if err != nil || len(menuMap) == 0 {
			t.Fatalf("failed to get menu: %v", err)
		}

		item, err := repo.GetByID(ctx, 1)
		if err != nil || item.Name != "Masala Tea" {
			t.Errorf("expected Masala Tea, got %v", item)
		}
	})

	t.Run("User Management", func(t *testing.T) {
		salt := repository.GenerateSalt()
		hash := repository.HashPassword("Password@123", salt)

		user, err := repo.CreateUser(ctx, models.User{
			Name:         "Test User",
			Email:        "testuser@example.com",
			MobileNumber: "9876543210",
			PasswordHash: hash,
			Salt:         salt,
			Role:         "client",
		})
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		fetched, err := repo.GetUserByEmail(ctx, "testuser@example.com")
		if err != nil || fetched.ID != user.ID {
			t.Errorf("expected user ID %d, got %v", user.ID, fetched)
		}
	})

	t.Run("Order Management", func(t *testing.T) {
		order, err := repo.CreateOrder(ctx, models.Order{
			UserID:        1,
			CustomerName:  "Test Order",
			CustomerPhone: "9876543210",
			OrderType:     "Dine-in",
			TableNumber:   "Table 1",
			Items: []models.OrderItem{
				{MenuItemID: 1, ItemName: "Masala Tea", Quantity: 2, Price: 30},
			},
		})
		if err != nil {
			t.Fatalf("failed to create order: %v", err)
		}

		if order.Status != "Pending" {
			t.Errorf("expected Pending status, got %s", order.Status)
		}

		if err := repo.UpdateOrderStatus(ctx, order.ID, "Completed"); err != nil {
			t.Fatalf("failed to update status: %v", err)
		}

		updated, _ := repo.GetOrderByID(ctx, order.ID)
		if updated.Status != "Completed" {
			t.Errorf("expected Completed status, got %s", updated.Status)
		}
	})
}
