package repository

import (
	"context"

	"teachar.in/models"
)

// InventoryRepository defines storage operations for store inventory, equipment, and operating expenses.
type InventoryRepository interface {
	// Inventory Items & Assets
	GetAllInventoryItems(ctx context.Context) ([]models.InventoryItem, error)
	GetInventoryItemByID(ctx context.Context, id int64) (*models.InventoryItem, error)
	SaveInventoryItem(ctx context.Context, item models.InventoryItem) (*models.InventoryItem, error)
	UpdateStockQuantity(ctx context.Context, id int64, delta float64) error
	DeleteInventoryItem(ctx context.Context, id int64) error

	// Operating Expenses & Purchases
	GetAllExpenses(ctx context.Context) ([]models.ExpenseEntry, error)
	SaveExpense(ctx context.Context, expense models.ExpenseEntry) (*models.ExpenseEntry, error)
	DeleteExpense(ctx context.Context, id int64) error
}
