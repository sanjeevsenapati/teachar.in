package repository

import (
	"context"

	"teachar.in/models"
)

// MenuRepository defines the interface for menu data storage.
type MenuRepository interface {
	GetAll(ctx context.Context) (map[string][]models.MenuItem, error)
	GetByID(ctx context.Context, id int64) (*models.MenuItem, error)
	GetFeatured(ctx context.Context) ([]models.MenuItem, error)
	Create(ctx context.Context, item models.MenuItem) (*models.MenuItem, error)
	Update(ctx context.Context, item models.MenuItem) error
	Delete(ctx context.Context, id int64) error
	ToggleAvailability(ctx context.Context, id int64) error
}

// UserRepository defines the interface for user and session management.
type UserRepository interface {
	CreateUser(ctx context.Context, user models.User) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
	GetAllUsers(ctx context.Context) ([]models.User, error)
	CreateSession(ctx context.Context, session models.Session) error
	GetSession(ctx context.Context, token string) (*models.Session, error)
	DeleteSession(ctx context.Context, token string) error
}

// OrderRepository defines the interface for customer orders.
type OrderRepository interface {
	CreateOrder(ctx context.Context, order models.Order) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int64) ([]models.Order, error)
	GetAllOrders(ctx context.Context) ([]models.Order, error)
	GetOrderByID(ctx context.Context, id int64) (*models.Order, error)
	UpdateOrderStatus(ctx context.Context, id int64, status string) error
	UpdateOrderStatusWithStaff(ctx context.Context, id int64, status string, staffID int64, staffName string, cancellationReason string) error
}

// AuditRepository defines the interface for audit trail logs.
type AuditRepository interface {
	CreateAuditLog(ctx context.Context, log models.AuditLog) (*models.AuditLog, error)
	GetAllAuditLogs(ctx context.Context) ([]models.AuditLog, error)
}