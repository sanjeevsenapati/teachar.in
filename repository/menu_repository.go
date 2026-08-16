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
	UpdateUser(ctx context.Context, user models.User) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
	GetAllUsers(ctx context.Context) ([]models.User, error)
	DeleteUser(ctx context.Context, id int64) error
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
	AssignOrderToStaff(ctx context.Context, id int64, staffID int64, staffName string, assignedBy string, estimatedMinutes int) error
	GetCancellationReasons(ctx context.Context) ([]string, error)
	AddCancellationReason(ctx context.Context, reason string) error
	DeleteCancellationReason(ctx context.Context, reason string) error
	SaveOrderReview(ctx context.Context, id int64, rating int, review string) error
}

// AuditRepository defines the interface for audit trail logs.
type AuditRepository interface {
	CreateAuditLog(ctx context.Context, log models.AuditLog) (*models.AuditLog, error)
	GetAllAuditLogs(ctx context.Context) ([]models.AuditLog, error)
}

// CouponRepository defines the interface for offer discount coupons.
type CouponRepository interface {
	CreateCoupon(ctx context.Context, coupon models.Coupon) (*models.Coupon, error)
	GetCouponByCode(ctx context.Context, code string) (*models.Coupon, error)
	GetAllCoupons(ctx context.Context) ([]models.Coupon, error)
	MarkCouponUsed(ctx context.Context, code string, orderID int64) error
	DeleteCoupon(ctx context.Context, id int64) error
}

// CafeSettingsRepository defines the interface for store & announcement bar settings.
type CafeSettingsRepository interface {
	GetCafeSettings(ctx context.Context) (*models.CafeSettings, error)
	UpdateCafeSettings(ctx context.Context, settings models.CafeSettings) error
}