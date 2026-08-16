package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"teachar.in/models"
)

func TestMultiFileRepository(t *testing.T) {
	tempDir := t.TempDir()

	repo, err := NewMultiFileRepository(tempDir)
	if err != nil {
		t.Fatalf("Failed creating multi-file repository: %v", err)
	}

	ctx := context.Background()

	// Test Users
	user, err := repo.CreateUser(ctx, models.User{
		Name:  "Test Concurrency User",
		Email: "testconcurrent@teachar.in",
	})
	if err != nil {
		t.Fatalf("Failed creating user: %v", err)
	}

	fetchedUser, err := repo.GetUserByEmail(ctx, "testconcurrent@teachar.in")
	if err != nil || fetchedUser.ID != user.ID {
		t.Fatalf("Failed fetching user by email: %v", err)
	}

	// Test Orders
	order, err := repo.CreateOrder(ctx, models.Order{
		UserID:       user.ID,
		CustomerName: user.Name,
		OrderType:    "Dine-in",
		TableNumber:  "Table 1",
		Items: []models.OrderItem{
			{MenuItemID: 1, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("Failed creating order: %v", err)
	}

	if order.ID == 0 {
		t.Fatalf("Expected valid order ID, got 0")
	}
}

func TestConcurrentLoginsAndOrders(t *testing.T) {
	tempDir := t.TempDir()

	repo, err := NewMultiFileRepository(tempDir)
	if err != nil {
		t.Fatalf("Failed creating multi-file repository: %v", err)
	}

	ctx := context.Background()
	const concurrentUsers = 100

	var wg sync.WaitGroup
	wg.Add(concurrentUsers)

	// Simulate 100 users logging in and placing orders simultaneously
	for i := 0; i < concurrentUsers; i++ {
		go func(idx int) {
			defer wg.Done()

			email := fmt.Sprintf("user%d@teachar.in", idx)
			u, err := repo.CreateUser(ctx, models.User{
				Name:  fmt.Sprintf("User %d", idx),
				Email: email,
			})
			if err != nil {
				t.Errorf("Concurrent user creation failed for %s: %v", email, err)
				return
			}

			// Create Session
			sessErr := repo.CreateSession(ctx, models.Session{
				ID:     fmt.Sprintf("token_%d", idx),
				UserID: u.ID,
			})
			if sessErr != nil {
				t.Errorf("Concurrent session creation failed for %s: %v", email, sessErr)
				return
			}

			// Create Order
			_, orderErr := repo.CreateOrder(ctx, models.Order{
				UserID:       u.ID,
				CustomerName: u.Name,
				OrderType:    "Takeaway",
				Items: []models.OrderItem{
					{MenuItemID: 1, Quantity: 1},
				},
			})
			if orderErr != nil {
				t.Errorf("Concurrent order creation failed for %s: %v", email, orderErr)
				return
			}
		}(i)
	}

	wg.Wait()

	// Verify all 100 users and orders exist
	users, err := repo.GetAllUsers(ctx)
	if err != nil {
		t.Fatalf("Failed getting all users: %v", err)
	}

	// 4 default seeded users + 100 concurrent users = 104
	if len(users) != concurrentUsers+4 {
		t.Fatalf("Expected %d users, got %d", concurrentUsers+4, len(users))
	}

	orders, err := repo.GetAllOrders(ctx)
	if err != nil {
		t.Fatalf("Failed getting all orders: %v", err)
	}

	if len(orders) != concurrentUsers {
		t.Fatalf("Expected %d orders, got %d", concurrentUsers, len(orders))
	}
}

func TestMigrationFromLegacyDB(t *testing.T) {
	tempDir := t.TempDir()

	legacyJSON := `{
  "next_user_id": 2,
  "next_menu_item_id": 2,
  "next_order_id": 2,
  "next_audit_log_id": 2,
  "users": [{"id": 1, "name": "Migrated User", "email": "migrated@teachar.in"}],
  "sessions": [],
  "menu_items": [{"id": 1, "name": "Migrated Tea", "category": "Tea", "price": 20}],
  "orders": [],
  "audit_logs": [],
  "cancellation_reasons": ["Item out of stock"]
}`
	_ = os.WriteFile(filepath.Join(tempDir, "db.json"), []byte(legacyJSON), 0644)

	repo, err := NewMultiFileRepository(tempDir)
	if err != nil {
		t.Fatalf("Failed initializing repository with legacy db.json: %v", err)
	}

	u, err := repo.GetUserByEmail(context.Background(), "migrated@teachar.in")
	if err != nil || u.Name != "Migrated User" {
		t.Fatalf("Migration failed, expected Migrated User, got: %v", err)
	}
}
