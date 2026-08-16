package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"teachar.in/models"
)

type dbSchema struct {
	NextUserID          int64                  `json:"next_user_id"`
	NextMenuItemID      int64                  `json:"next_menu_item_id"`
	NextOrderID         int64                  `json:"next_order_id"`
	NextAuditLogID      int64                  `json:"next_audit_log_id"`
	NextCouponID        int64                  `json:"next_coupon_id"`
	NextInventoryID     int64                  `json:"next_inventory_id"`
	NextExpenseID       int64                  `json:"next_expense_id"`
	Users               []models.User          `json:"users"`
	Sessions            []models.Session       `json:"sessions"`
	MenuItems           []models.MenuItem      `json:"menu_items"`
	Orders              []models.Order         `json:"orders"`
	AuditLogs           []models.AuditLog      `json:"audit_logs"`
	Coupons             []models.Coupon        `json:"coupons"`
	InventoryItems      []models.InventoryItem `json:"inventory_items"`
	Expenses            []models.ExpenseEntry  `json:"expenses"`
	CancellationReasons []string               `json:"cancellation_reasons"`
}

// JSONRepository is a thread-safe, file-backed database using only standard library packages.
type JSONRepository struct {
	mu       sync.RWMutex
	filePath string
	data     dbSchema
}

// Helper to hash password using stdlib sha256 + salt
func HashPassword(password, salt string) string {
	hash := sha256.Sum256([]byte(password + salt))
	return hex.EncodeToString(hash[:])
}

// GenerateSalt creates a random hex salt using crypto/rand
func GenerateSalt() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// NewJSONRepository creates or loads the JSON file store with initial seed data.
func NewJSONRepository(filePath string) (*JSONRepository, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	repo := &JSONRepository{
		filePath: filePath,
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		repo.seedDefaultData()
		if err := repo.save(); err != nil {
			return nil, fmt.Errorf("failed to save initial seed: %w", err)
		}
	} else {
		if err := repo.load(); err != nil {
			return nil, fmt.Errorf("failed to load database file: %w", err)
		}
	}

	return repo, nil
}

func (r *JSONRepository) load() error {
	bytes, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, &r.data)
}

func (r *JSONRepository) save() error {
	bytes, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := r.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, bytes, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, r.filePath)
}

func (r *JSONRepository) seedDefaultData() {
	superAdminSalt := GenerateSalt()
	adminSalt := GenerateSalt()
	clientSalt := GenerateSalt()
	staffSalt := GenerateSalt()

	r.data = dbSchema{
		NextUserID:     5,
		NextMenuItemID: 14,
		NextOrderID:    1,
		NextAuditLogID: 2,
		NextCouponID:   1,
		Users: []models.User{
			{
				ID:           1,
				Name:         "Super Admin",
				Email:        "superadmin@teachar.in",
				MobileNumber: "9000000000",
				PasswordHash: HashPassword("SuperAdmin@123", superAdminSalt),
				Salt:         superAdminSalt,
				Role:         "superadmin",
				CreatedAt:    time.Now(),
			},
			{
				ID:           2,
				Name:         "TEACHAR Admin",
				Email:        "admin@teachar.in",
				MobileNumber: "9876543210",
				PasswordHash: HashPassword("Admin@123", adminSalt),
				Salt:         adminSalt,
				Role:         "admin",
				CreatedAt:    time.Now(),
			},
			{
				ID:           3,
				Name:         "Sample Client",
				Email:        "client@teachar.in",
				MobileNumber: "9123456789",
				PasswordHash: HashPassword("Client@123", clientSalt),
				Salt:         clientSalt,
				Role:         "client",
				CreatedAt:    time.Now(),
			},
			{
				ID:           4,
				Name:         "Counter Staff",
				Email:        "staff@teachar.in",
				MobileNumber: "9555544444",
				PasswordHash: HashPassword("Staff@123", staffSalt),
				Salt:         staffSalt,
				Role:         "staff",
				CreatedAt:    time.Now(),
			},
		},
		Sessions: []models.Session{},
		MenuItems: []models.MenuItem{
			{ID: 1, Name: "Masala Tea", Description: "A classic blend of black tea and aromatic spices.", Category: "Tea", Price: 30, Image: "/static/images/masala-tea.jpg", Available: true},
			{ID: 2, Name: "Ginger Tea", Description: "Zesty and refreshing tea with a ginger kick.", Category: "Tea", Price: 30, Image: "/static/images/ginger-tea.jpg", Available: true},
			{ID: 3, Name: "Lemon Tea", Description: "Light and tangy, perfect for a fresh start.", Category: "Tea", Price: 30, Image: "/static/images/lemon-tea.jpg", Available: true},
			{ID: 4, Name: "Cold Coffee", Description: "Rich, creamy, and perfectly chilled.", Category: "Coffee", Price: 80, Image: "/static/images/cold-coffee.jpg", Available: true},
			{ID: 5, Name: "Hot Coffee", Description: "A strong and aromatic freshly brewed coffee.", Category: "Coffee", Price: 40, Image: "/static/images/hot-coffee.jpg", Available: true},
			{ID: 6, Name: "Veg Sandwich", Description: "A wholesome sandwich with fresh vegetables.", Category: "Snacks", Price: 70, Image: "/static/images/veg-sandwich.jpg", Available: true},
			{ID: 7, Name: "Samosa", Description: "Crispy pastry filled with spiced potatoes.", Category: "Snacks", Price: 25, Image: "/static/images/samosa.jpg", Available: true},
			{ID: 8, Name: "Coke", Description: "Chilled Coca-Cola.", Category: "Cold Drinks", Price: 40, Image: "/static/images/coke.jpg", Available: true},
			{ID: 9, Name: "Red Tea", Description: "Fragrant and antioxidant-rich crimson herbal tea brew.", Category: "Tea", Price: 35, Image: "/static/images/red-tea.jpg", Available: true},
			{ID: 10, Name: "Popped Rice", Description: "Crispy roasted puffed rice tossed with peanuts, spices, and fresh herbs.", Category: "Snacks", Price: 30, Image: "/static/images/popped-rice.jpg", Available: true},
			{ID: 11, Name: "Vanilla Popped Rice", Description: "Sweet and crunchy puffed rice delicately infused with vanilla bean and honey glaze.", Category: "Snacks", Price: 35, Image: "/static/images/vanilla-popped-rice.jpg", Available: true},
			{ID: 12, Name: "Red Tea & Popped Rice Combo", Description: "A comforting cup of crimson Red Tea served with a separate small sachet of crispy Popped Rice to sprinkle on top.", Category: "Tea", Price: 50, Image: "/static/images/red-tea-combo.jpg", Available: true},
			{ID: 13, Name: "Milk Tea with Overnight High fiber Roti", Description: "Traditional hot milk tea served alongside authentic overnight 1-day old fermented Basi Roti.", Category: "Tea", Price: 45, Image: "/static/images/milk-tea-basi-roti.jpg", Available: true},
		},
		Orders: []models.Order{},
		AuditLogs: []models.AuditLog{
			{
				ID:        1,
				Timestamp: time.Now(),
				ActorID:   1,
				ActorName: "Super Admin",
				ActorRole: "superadmin",
				Action:    "SYSTEM_INIT",
				Details:   "Initialized system with default Superadmin, Admin, Staff, and Client accounts.",
				IPAddress: "127.0.0.1",
			},
		},
	}
}

// --- MenuRepository Implementation ---

func (r *JSONRepository) GetAll(ctx context.Context) (map[string][]models.MenuItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]models.MenuItem)
	for _, item := range r.data.MenuItems {
		result[item.Category] = append(result[item.Category], item)
	}
	return result, nil
}

func (r *JSONRepository) GetByID(ctx context.Context, id int64) (*models.MenuItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, item := range r.data.MenuItems {
		if item.ID == id {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (r *JSONRepository) GetFeatured(ctx context.Context) ([]models.MenuItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var featured []models.MenuItem
	for i, item := range r.data.MenuItems {
		if i < 4 {
			featured = append(featured, item)
		} else {
			break
		}
	}
	return featured, nil
}

func (r *JSONRepository) Create(ctx context.Context, item models.MenuItem) (*models.MenuItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item.ID = r.data.NextMenuItemID
	r.data.NextMenuItemID++
	r.data.MenuItems = append(r.data.MenuItems, item)

	if err := r.save(); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *JSONRepository) Update(ctx context.Context, item models.MenuItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, existing := range r.data.MenuItems {
		if existing.ID == item.ID {
			r.data.MenuItems[i] = item
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("item not found")
	}
	return r.save()
}

func (r *JSONRepository) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	newItems := make([]models.MenuItem, 0, len(r.data.MenuItems))
	found := false
	for _, item := range r.data.MenuItems {
		if item.ID == id {
			found = true
			continue
		}
		newItems = append(newItems, item)
	}
	if !found {
		return fmt.Errorf("item not found")
	}
	r.data.MenuItems = newItems
	return r.save()
}

func (r *JSONRepository) ToggleAvailability(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, item := range r.data.MenuItems {
		if item.ID == id {
			r.data.MenuItems[i].Available = !r.data.MenuItems[i].Available
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("item not found")
	}
	return r.save()
}

// --- UserRepository Implementation ---

func (r *JSONRepository) CreateUser(ctx context.Context, user models.User) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.data.Users {
		if u.Email == user.Email {
			return nil, fmt.Errorf("user with this email already exists")
		}
	}

	user.ID = r.data.NextUserID
	r.data.NextUserID++
	user.CreatedAt = time.Now()
	r.data.Users = append(r.data.Users, user)

	if err := r.save(); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *JSONRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.data.Users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (r *JSONRepository) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.data.Users {
		if u.ID == id {
			return &u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (r *JSONRepository) CreateSession(ctx context.Context, session models.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data.Sessions = append(r.data.Sessions, session)
	return r.save()
}

func (r *JSONRepository) GetSession(ctx context.Context, token string) (*models.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	for _, s := range r.data.Sessions {
		if s.ID == token {
			if now.After(s.ExpiresAt) {
				return nil, fmt.Errorf("session expired")
			}
			return &s, nil
		}
	}
	return nil, fmt.Errorf("session not found")
}

func (r *JSONRepository) DeleteSession(ctx context.Context, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	newSessions := make([]models.Session, 0, len(r.data.Sessions))
	for _, s := range r.data.Sessions {
		if s.ID != token {
			newSessions = append(newSessions, s)
		}
	}
	r.data.Sessions = newSessions
	return r.save()
}

// --- OrderRepository Implementation ---

func (r *JSONRepository) CreateOrder(ctx context.Context, order models.Order) (*models.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	order.ID = r.data.NextOrderID
	r.data.NextOrderID++
	order.Status = "Pending"
	order.CreatedAt = time.Now()

	for i := range order.Items {
		order.Items[i].ID = int64(i + 1)
		order.Items[i].OrderID = order.ID
	}

	r.data.Orders = append(r.data.Orders, order)
	if err := r.save(); err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *JSONRepository) GetOrdersByUserID(ctx context.Context, userID int64) ([]models.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []models.Order
	for i := len(r.data.Orders) - 1; i >= 0; i-- {
		if r.data.Orders[i].UserID == userID {
			result = append(result, r.data.Orders[i])
		}
	}
	return result, nil
}

func (r *JSONRepository) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []models.Order
	for i := len(r.data.Orders) - 1; i >= 0; i-- {
		result = append(result, r.data.Orders[i])
	}
	return result, nil
}

func (r *JSONRepository) GetOrderByID(ctx context.Context, id int64) (*models.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, o := range r.data.Orders {
		if o.ID == id {
			return &o, nil
		}
	}
	return nil, fmt.Errorf("order not found")
}

func (r *JSONRepository) UpdateOrderStatus(ctx context.Context, id int64, status string) error {
	return r.UpdateOrderStatusWithStaff(ctx, id, status, 0, "", "")
}

func (r *JSONRepository) UpdateOrderStatusWithStaff(ctx context.Context, id int64, status string, staffID int64, staffName string, cancellationReason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, o := range r.data.Orders {
		if o.ID == id {
			r.data.Orders[i].Status = status
			if staffID != 0 {
				r.data.Orders[i].AssignedStaffID = staffID
				r.data.Orders[i].AssignedStaffName = staffName
				if r.data.Orders[i].AssignedBy == "" {
					r.data.Orders[i].AssignedBy = "Self Claimed"
				}
				if r.data.Orders[i].AssignedAt == nil {
					now := time.Now()
					r.data.Orders[i].AssignedAt = &now
				}
			}
			if cancellationReason != "" {
				r.data.Orders[i].CancellationReason = cancellationReason
			}
			if status == "Completed" && r.data.Orders[i].CompletedAt == nil {
				now := time.Now()
				r.data.Orders[i].CompletedAt = &now
				duration := int(now.Sub(r.data.Orders[i].CreatedAt).Minutes())
				if duration < 1 {
					duration = 1
				}
				r.data.Orders[i].FulfillmentMinutes = duration
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("order not found")
	}
	return r.save()
}

func (r *JSONRepository) AssignOrderToStaff(ctx context.Context, id int64, staffID int64, staffName string, assignedBy string, estimatedMinutes int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, o := range r.data.Orders {
		if o.ID == id {
			now := time.Now()
			r.data.Orders[i].AssignedStaffID = staffID
			r.data.Orders[i].AssignedStaffName = staffName
			r.data.Orders[i].AssignedBy = assignedBy
			r.data.Orders[i].AssignedAt = &now
			if estimatedMinutes > 0 {
				r.data.Orders[i].EstimatedMinutes = estimatedMinutes
			} else if r.data.Orders[i].EstimatedMinutes <= 0 {
				r.data.Orders[i].EstimatedMinutes = 20
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("order not found")
	}
	return r.save()
}

func (r *JSONRepository) GetAllUsers(ctx context.Context) ([]models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []models.User
	for i := len(r.data.Users) - 1; i >= 0; i-- {
		result = append(result, r.data.Users[i])
	}
	return result, nil
}

// --- AuditRepository Implementation ---

func (r *JSONRepository) CreateAuditLog(ctx context.Context, log models.AuditLog) (*models.AuditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	log.ID = r.data.NextAuditLogID
	r.data.NextAuditLogID++
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	r.data.AuditLogs = append(r.data.AuditLogs, log)
	if err := r.save(); err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *JSONRepository) GetAllAuditLogs(ctx context.Context) ([]models.AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []models.AuditLog
	for i := len(r.data.AuditLogs) - 1; i >= 0; i-- {
		result = append(result, r.data.AuditLogs[i])
	}
	return result, nil
}

// --- Cancellation Reasons Management ---

func (r *JSONRepository) GetCancellationReasons(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defaults := []string{
		"Item out of stock",
		"Kitchen load exceeded / High volume",
		"Customer requested cancellation",
		"Raw ingredients unavailable",
		"End of operational hours / Store closing",
		"Delivery location unreachable",
	}

	if len(r.data.CancellationReasons) == 0 {
		return defaults, nil
	}

	result := make([]string, len(r.data.CancellationReasons))
	copy(result, r.data.CancellationReasons)
	return result, nil
}

func (r *JSONRepository) AddCancellationReason(ctx context.Context, reason string) error {
	if reason == "" {
		return errors.New("cancellation reason cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.data.CancellationReasons) == 0 {
		r.data.CancellationReasons = []string{
			"Item out of stock",
			"Kitchen load exceeded / High volume",
			"Customer requested cancellation",
			"Raw ingredients unavailable",
			"End of operational hours / Store closing",
			"Delivery location unreachable",
		}
	}

	for _, existing := range r.data.CancellationReasons {
		if existing == reason {
			return nil
		}
	}

	r.data.CancellationReasons = append(r.data.CancellationReasons, reason)
	return r.save()
}

func (r *JSONRepository) DeleteCancellationReason(ctx context.Context, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.data.CancellationReasons) == 0 {
		r.data.CancellationReasons = []string{
			"Item out of stock",
			"Kitchen load exceeded / High volume",
			"Customer requested cancellation",
			"Raw ingredients unavailable",
			"End of operational hours / Store closing",
			"Delivery location unreachable",
		}
	}

	var updated []string
	for _, existing := range r.data.CancellationReasons {
		if existing != reason {
			updated = append(updated, existing)
		}
	}
	r.data.CancellationReasons = updated
	return r.save()
}

func (r *JSONRepository) SaveOrderReview(ctx context.Context, id int64, rating int, review string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, o := range r.data.Orders {
		if o.ID == id {
			r.data.Orders[i].Rating = rating
			r.data.Orders[i].Review = review
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("order not found")
	}
	return r.save()
}

// --- CouponRepository Implementation ---

func (r *JSONRepository) CreateCoupon(ctx context.Context, coupon models.Coupon) (*models.Coupon, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	codeUpper := strings.ToUpper(strings.TrimSpace(coupon.Code))
	if codeUpper == "" {
		return nil, errors.New("coupon code cannot be empty")
	}
	for _, c := range r.data.Coupons {
		if strings.ToUpper(c.Code) == codeUpper {
			return nil, fmt.Errorf("coupon code '%s' already exists", codeUpper)
		}
	}

	coupon.ID = r.data.NextCouponID
	r.data.NextCouponID++
	coupon.Code = codeUpper
	coupon.CreatedAt = time.Now()
	coupon.IsUsed = false

	r.data.Coupons = append(r.data.Coupons, coupon)
	if err := r.save(); err != nil {
		return nil, err
	}
	return &coupon, nil
}

func (r *JSONRepository) GetCouponByCode(ctx context.Context, code string) (*models.Coupon, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	codeUpper := strings.ToUpper(strings.TrimSpace(code))
	for _, c := range r.data.Coupons {
		if strings.ToUpper(c.Code) == codeUpper {
			return &c, nil
		}
	}
	return nil, errors.New("coupon not found")
}

func (r *JSONRepository) GetAllCoupons(ctx context.Context) ([]models.Coupon, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.Coupon, len(r.data.Coupons))
	copy(result, r.data.Coupons)
	return result, nil
}

func (r *JSONRepository) MarkCouponUsed(ctx context.Context, code string, orderID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	codeUpper := strings.ToUpper(strings.TrimSpace(code))
	found := false
	for i, c := range r.data.Coupons {
		if strings.ToUpper(c.Code) == codeUpper {
			if c.IsUsed {
				return errors.New("coupon has already been used")
			}
			now := time.Now()
			r.data.Coupons[i].IsUsed = true
			r.data.Coupons[i].UsedAt = &now
			r.data.Coupons[i].UsedByOrderID = orderID
			found = true
			break
		}
	}
	if !found {
		return errors.New("coupon not found")
	}
	return r.save()
}

func (r *JSONRepository) DeleteCoupon(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var updated []models.Coupon
	found := false
	for _, c := range r.data.Coupons {
		if c.ID == id {
			found = true
			continue
		}
		updated = append(updated, c)
	}
	if !found {
		return errors.New("coupon not found")
	}
	r.data.Coupons = updated
	return r.save()
}

func (r *JSONRepository) GetAllInventoryItems(ctx context.Context) ([]models.InventoryItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.InventoryItem, len(r.data.InventoryItems))
	copy(result, r.data.InventoryItems)
	return result, nil
}

func (r *JSONRepository) GetInventoryItemByID(ctx context.Context, id int64) (*models.InventoryItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, item := range r.data.InventoryItems {
		if item.ID == id {
			return &item, nil
		}
	}
	return nil, errors.New("inventory item not found")
}

func (r *JSONRepository) SaveInventoryItem(ctx context.Context, item models.InventoryItem) (*models.InventoryItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	item.UpdatedAt = now
	item.TotalValue = item.StockQuantity * item.UnitCost

	if item.Category == "Equipment" || item.Category == "Furniture" {
		item.Status = "Active Asset"
	} else if item.StockQuantity <= 0 {
		item.Status = "Out of Stock"
	} else if item.StockQuantity <= item.ReorderLevel {
		item.Status = "Low Stock"
	} else {
		item.Status = "In Stock"
	}

	if item.ID == 0 {
		if r.data.NextInventoryID <= 0 {
			r.data.NextInventoryID = 1
		}
		item.ID = r.data.NextInventoryID
		r.data.NextInventoryID++
		r.data.InventoryItems = append(r.data.InventoryItems, item)
		if err := r.save(); err != nil {
			return nil, err
		}
		return &r.data.InventoryItems[len(r.data.InventoryItems)-1], nil
	}

	for i, existing := range r.data.InventoryItems {
		if existing.ID == item.ID {
			r.data.InventoryItems[i] = item
			if err := r.save(); err != nil {
				return nil, err
			}
			return &r.data.InventoryItems[i], nil
		}
	}
	return nil, errors.New("inventory item not found")
}

func (r *JSONRepository) UpdateStockQuantity(ctx context.Context, id int64, delta float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, item := range r.data.InventoryItems {
		if item.ID == id {
			r.data.InventoryItems[i].StockQuantity += delta
			if r.data.InventoryItems[i].StockQuantity < 0 {
				r.data.InventoryItems[i].StockQuantity = 0
			}
			r.data.InventoryItems[i].TotalValue = r.data.InventoryItems[i].StockQuantity * r.data.InventoryItems[i].UnitCost
			r.data.InventoryItems[i].UpdatedAt = time.Now()

			if r.data.InventoryItems[i].StockQuantity <= 0 {
				r.data.InventoryItems[i].Status = "Out of Stock"
			} else if r.data.InventoryItems[i].StockQuantity <= r.data.InventoryItems[i].ReorderLevel {
				r.data.InventoryItems[i].Status = "Low Stock"
			} else {
				r.data.InventoryItems[i].Status = "In Stock"
			}
			return r.save()
		}
	}
	return errors.New("inventory item not found")
}

func (r *JSONRepository) DeleteInventoryItem(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var updated []models.InventoryItem
	for _, item := range r.data.InventoryItems {
		if item.ID != id {
			updated = append(updated, item)
		}
	}
	r.data.InventoryItems = updated
	return r.save()
}

func (r *JSONRepository) GetAllExpenses(ctx context.Context) ([]models.ExpenseEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.ExpenseEntry, len(r.data.Expenses))
	copy(result, r.data.Expenses)
	return result, nil
}

func (r *JSONRepository) SaveExpense(ctx context.Context, expense models.ExpenseEntry) (*models.ExpenseEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	expense.CreatedAt = now
	if expense.ExpenseDate.IsZero() {
		expense.ExpenseDate = now
	}
	if expense.TotalAmount <= 0 && expense.Quantity > 0 && expense.UnitPrice > 0 {
		expense.TotalAmount = expense.Quantity * expense.UnitPrice
	}

	if expense.ID == 0 {
		if r.data.NextExpenseID <= 0 {
			r.data.NextExpenseID = 1
		}
		expense.ID = r.data.NextExpenseID
		r.data.NextExpenseID++
		r.data.Expenses = append(r.data.Expenses, expense)
		if err := r.save(); err != nil {
			return nil, err
		}
		return &r.data.Expenses[len(r.data.Expenses)-1], nil
	}

	for i, exp := range r.data.Expenses {
		if exp.ID == expense.ID {
			r.data.Expenses[i] = expense
			if err := r.save(); err != nil {
				return nil, err
			}
			return &r.data.Expenses[i], nil
		}
	}
	return nil, errors.New("expense entry not found")
}

func (r *JSONRepository) DeleteExpense(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var updated []models.ExpenseEntry
	for _, exp := range r.data.Expenses {
		if exp.ID != id {
			updated = append(updated, exp)
		}
	}
	r.data.Expenses = updated
	return r.save()
}
