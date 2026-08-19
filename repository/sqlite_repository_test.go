package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"teachar.in/models"
)

func TestSQLiteRepository_BasicCRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	repo, err := NewSQLiteRepository(dbPath, tempDir)
	if err != nil {
		t.Fatalf("Failed creating sqlite repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// 1. Test Users
	u, err := repo.CreateUser(ctx, models.User{
		Name:         "SQLite Test User",
		Email:        "sqliteuser@teachar.in",
		MobileNumber: "9998887776",
		Role:         "client",
		Status:       "Active",
	})
	if err != nil {
		t.Fatalf("Failed creating user: %v", err)
	}

	fetchedUser, err := repo.GetUserByEmail(ctx, "sqliteuser@teachar.in")
	if err != nil || fetchedUser.ID != u.ID {
		t.Fatalf("Failed getting user by email: %v", err)
	}

	fetchedByID, err := repo.GetUserByID(ctx, u.ID)
	if err != nil || fetchedByID.Name != "SQLite Test User" {
		t.Fatalf("Failed getting user by ID: %v", err)
	}

	// 2. Test Menu
	menuMap, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed getting menu: %v", err)
	}
	if len(menuMap) == 0 {
		t.Fatalf("Expected seeded menu items, got 0")
	}

	newItem, err := repo.Create(ctx, models.MenuItem{
		Name:        "Test SQLite Herbal Chai",
		Description: "Herbal and fresh",
		Category:    "Tea",
		Price:       45,
		Available:   true,
	})
	if err != nil {
		t.Fatalf("Failed creating menu item: %v", err)
	}

	itemByID, err := repo.GetByID(ctx, newItem.ID)
	if err != nil || itemByID.Name != newItem.Name {
		t.Fatalf("Failed getting menu item by ID: %v", err)
	}

	// 3. Test Orders
	order, err := repo.CreateOrder(ctx, models.Order{
		UserID:        u.ID,
		CustomerName:  u.Name,
		CustomerPhone: u.MobileNumber,
		OrderType:     "Dine-in",
		TableNumber:   "Table 5",
		TotalPrice:    90,
		Items: []models.OrderItem{
			{MenuItemID: newItem.ID, ItemName: newItem.Name, Quantity: 2, Price: 45},
		},
	})
	if err != nil {
		t.Fatalf("Failed creating order: %v", err)
	}
	if order.ID == 0 {
		t.Fatalf("Expected non-zero order ID")
	}

	userOrders, err := repo.GetOrdersByUserID(ctx, u.ID)
	if err != nil || len(userOrders) == 0 {
		t.Fatalf("Failed getting user orders: %v", err)
	}
	if len(userOrders[0].Items) != 1 {
		t.Fatalf("Expected 1 order item, got %d", len(userOrders[0].Items))
	}

	// 4. Test Coupons
	coupon, err := repo.CreateCoupon(ctx, models.Coupon{
		Code:          "SQLITETEST10",
		DiscountType:  "percentage",
		DiscountValue: 10,
		ExpiryDate:    time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Failed creating coupon: %v", err)
	}

	cByCode, err := repo.GetCouponByCode(ctx, "SQLITETEST10")
	if err != nil || cByCode.ID != coupon.ID {
		t.Fatalf("Failed getting coupon by code: %v", err)
	}

	// 5. Test Inventory & Expenses
	inv, err := repo.SaveInventoryItem(ctx, models.InventoryItem{
		Category:      "Raw Material",
		ItemName:      "Cardamom",
		Unit:          "kg",
		StockQuantity: 10,
		ReorderLevel:  2,
		UnitCost:      1500,
	})
	if err != nil {
		t.Fatalf("Failed saving inventory item: %v", err)
	}
	if inv.ID == 0 || inv.TotalValue != 15000 {
		t.Fatalf("Unexpected inventory item: %+v", inv)
	}

	exp, err := repo.SaveExpense(ctx, models.ExpenseEntry{
		Category:      "Raw Materials",
		Title:         "Cardamom Purchase",
		TotalAmount:   15000,
		PaymentMethod: "UPI",
	})
	if err != nil || exp.ID == 0 {
		t.Fatalf("Failed saving expense: %v", err)
	}

	// 6. Test Subscriptions
	sub, err := repo.SaveSubscription(ctx, models.UserSubscription{
		UserID:            u.ID,
		UserName:          u.Name,
		UserEmail:         u.Email,
		TierID:            "gold",
		TierName:          "Gold Pass",
		PricePaid:         999,
		DiscountPercent:   15,
		DailyFreeCupLimit: 1,
		Status:            "Active",
	})
	if err != nil || sub.ID == 0 {
		t.Fatalf("Failed saving subscription: %v", err)
	}

	userSub, err := repo.GetSubscriptionByUserID(ctx, u.ID)
	if err != nil || userSub.ID != sub.ID {
		t.Fatalf("Failed getting active user subscription: %v", err)
	}

	// 7. Test Cafe Settings
	settings, err := repo.GetCafeSettings(ctx)
	if err != nil || settings.StoreName == "" {
		t.Fatalf("Failed getting cafe settings: %v", err)
	}
	settings.StoreName = "Updated Sanctuary"
	if err := repo.UpdateCafeSettings(ctx, *settings); err != nil {
		t.Fatalf("Failed updating cafe settings: %v", err)
	}
	updatedSettings, err := repo.GetCafeSettings(ctx)
	if err != nil || updatedSettings.StoreName != "Updated Sanctuary" {
		t.Fatalf("Expected updated store name, got: %s", updatedSettings.StoreName)
	}
}

func TestSQLiteRepository_Concurrency(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "concurrent_test.db")

	repo, err := NewSQLiteRepository(dbPath, tempDir)
	if err != nil {
		t.Fatalf("Failed creating sqlite repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	const concurrentUsers = 50

	var wg sync.WaitGroup
	wg.Add(concurrentUsers)

	for i := 0; i < concurrentUsers; i++ {
		go func(idx int) {
			defer wg.Done()

			email := fmt.Sprintf("sqlite_user%d@teachar.in", idx)
			u, err := repo.CreateUser(ctx, models.User{
				Name:  fmt.Sprintf("Concurrent User %d", idx),
				Email: email,
				Role:  "client",
			})
			if err != nil {
				t.Errorf("Concurrent user creation failed for %s: %v", email, err)
				return
			}

			sessErr := repo.CreateSession(ctx, models.Session{
				ID:        fmt.Sprintf("sql_token_%d", idx),
				UserID:    u.ID,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			})
			if sessErr != nil {
				t.Errorf("Concurrent session creation failed: %v", sessErr)
				return
			}

			_, orderErr := repo.CreateOrder(ctx, models.Order{
				UserID:       u.ID,
				CustomerName: u.Name,
				OrderType:    "Takeaway",
				TotalPrice:   30,
				Items: []models.OrderItem{
					{MenuItemID: 1, ItemName: "Masala Tea", Quantity: 1, Price: 30},
				},
			})
			if orderErr != nil {
				t.Errorf("Concurrent order creation failed: %v", orderErr)
				return
			}
		}(i)
	}

	wg.Wait()

	users, err := repo.GetAllUsers(ctx)
	if err != nil {
		t.Fatalf("Failed getting all users: %v", err)
	}
	// 4 default seeded users + 50 concurrent users = 54
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

func TestSQLiteRepository_AutoMigrationFromJSON(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	_ = os.MkdirAll(dataDir, 0755)

	usersJSON := `{
		"users": [
			{"id": 10, "name": "Migrated SQLite User", "email": "migrated_sql@teachar.in", "role": "client", "password_hash": "hash", "salt": "salt"}
		]
	}`
	_ = os.WriteFile(filepath.Join(dataDir, "users.json"), []byte(usersJSON), 0644)

	menuJSON := `{
		"menu_items": [
			{"id": 20, "name": "Migrated Kashmiri Kahwa", "category": "Tea", "price": 55, "available": true}
		]
	}`
	_ = os.WriteFile(filepath.Join(dataDir, "menu.json"), []byte(menuJSON), 0644)

	dbPath := filepath.Join(tempDir, "migrated.db")
	repo, err := NewSQLiteRepository(dbPath, dataDir)
	if err != nil {
		t.Fatalf("Failed initializing repo with migration: %v", err)
	}
	defer repo.Close()

	u, err := repo.GetUserByEmail(context.Background(), "migrated_sql@teachar.in")
	if err != nil || u.Name != "Migrated SQLite User" {
		t.Fatalf("Migration user check failed: %v", err)
	}

	item, err := repo.GetByID(context.Background(), 20)
	if err != nil || item.Name != "Migrated Kashmiri Kahwa" {
		t.Fatalf("Migration menu item check failed: %v", err)
	}
}
