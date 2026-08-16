package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"teachar.in/models"
)

type usersSchema struct {
	NextUserID int64         `json:"next_user_id"`
	Users      []models.User `json:"users"`
}

type sessionsSchema struct {
	Sessions []models.Session `json:"sessions"`
}

type menuSchema struct {
	NextMenuItemID int64             `json:"next_menu_item_id"`
	MenuItems      []models.MenuItem `json:"menu_items"`
}

type ordersSchema struct {
	NextOrderID         int64          `json:"next_order_id"`
	CancellationReasons []string       `json:"cancellation_reasons"`
	Orders              []models.Order `json:"orders"`
}

type auditLogsSchema struct {
	NextAuditLogID int64             `json:"next_audit_log_id"`
	AuditLogs      []models.AuditLog `json:"audit_logs"`
}

type couponsSchema struct {
	NextCouponID int64           `json:"next_coupon_id"`
	Coupons      []models.Coupon `json:"coupons"`
}

type inventorySchema struct {
	NextInventoryID int64                  `json:"next_inventory_id"`
	InventoryItems  []models.InventoryItem `json:"inventory_items"`
}

type expensesSchema struct {
	NextExpenseID int64                 `json:"next_expense_id"`
	Expenses      []models.ExpenseEntry `json:"expenses"`
}

type apiKeysSchema struct {
	NextAPIKeyID int64          `json:"next_api_key_id"`
	APIKeys      []models.APIKey `json:"api_keys"`
}

type membershipsSchema struct {
	NextSubscriptionID int64                     `json:"next_subscription_id"`
	Subscriptions      []models.UserSubscription `json:"subscriptions"`
}

// MultiFileRepository provides domain-isolated, fine-grained locking, O(1) in-memory indexed storage using standard library packages.
type MultiFileRepository struct {
	dataDir string

	// Domain Mutexes for Concurrent Access
	usersMu       sync.RWMutex
	sessionsMu    sync.RWMutex
	menuMu        sync.RWMutex
	ordersMu      sync.RWMutex
	auditLogsMu   sync.RWMutex
	couponsMu     sync.RWMutex
	inventoryMu   sync.RWMutex
	expensesMu    sync.RWMutex
	apiKeysMu     sync.RWMutex
	membershipsMu sync.RWMutex
	settingsMu    sync.RWMutex

	// Data File Paths
	usersFile       string
	sessionsFile    string
	menuFile        string
	ordersFile      string
	auditLogsFile   string
	couponsFile     string
	inventoryFile   string
	expensesFile    string
	apiKeysFile     string
	membershipsFile string
	settingsFile    string

	// O(1) In-Memory Fast Lookup Maps & Caches
	usersByEmail          map[string]*models.User
	usersByID             map[int64]*models.User
	sessionsByToken       map[string]*models.Session
	menuItemsByID         map[int64]*models.MenuItem
	ordersByID            map[int64]*models.Order
	couponsByCode         map[string]*models.Coupon
	inventoryItemsByID    map[int64]*models.InventoryItem
	apiKeysByHash         map[string]*models.APIKey
	subscriptionsByUserID map[int64]*models.UserSubscription
	cafeSettings          *models.CafeSettings

	// Domain Data Arrays
	users               []models.User
	sessions            []models.Session
	menuItems           []models.MenuItem
	orders              []models.Order
	auditLogs           []models.AuditLog
	coupons             []models.Coupon
	cancellationReasons []string
	inventoryItems      []models.InventoryItem
	expenses            []models.ExpenseEntry
	apiKeys             []models.APIKey
	subscriptions       []models.UserSubscription

	// Atomic Sequence Counters for High-Concurrency ID Generation
	nextUserID         atomic.Int64
	nextMenuItemID     atomic.Int64
	nextOrderID        atomic.Int64
	nextAuditLogID     atomic.Int64
	nextCouponID       atomic.Int64
	nextInventoryID    atomic.Int64
	nextExpenseID      atomic.Int64
	nextAPIKeyID       atomic.Int64
	nextSubscriptionID atomic.Int64
}

// NewMultiFileRepository initializes and loads domain-isolated files, with automatic migration from old db.json if present.
func NewMultiFileRepository(dataDir string) (*MultiFileRepository, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	repo := &MultiFileRepository{
		dataDir:            dataDir,
		usersFile:          filepath.Join(dataDir, "users.json"),
		sessionsFile:       filepath.Join(dataDir, "sessions.json"),
		menuFile:           filepath.Join(dataDir, "menu.json"),
		ordersFile:         filepath.Join(dataDir, "orders.json"),
		auditLogsFile:      filepath.Join(dataDir, "audit_logs.json"),
		couponsFile:        filepath.Join(dataDir, "coupons.json"),
		inventoryFile:      filepath.Join(dataDir, "inventory.json"),
		expensesFile:          filepath.Join(dataDir, "expenses.json"),
		apiKeysFile:           filepath.Join(dataDir, "api_keys.json"),
		membershipsFile:       filepath.Join(dataDir, "memberships.json"),
		settingsFile:          filepath.Join(dataDir, "settings.json"),
		usersByEmail:          make(map[string]*models.User),
		usersByID:             make(map[int64]*models.User),
		sessionsByToken:       make(map[string]*models.Session),
		menuItemsByID:         make(map[int64]*models.MenuItem),
		ordersByID:            make(map[int64]*models.Order),
		couponsByCode:         make(map[string]*models.Coupon),
		inventoryItemsByID:    make(map[int64]*models.InventoryItem),
		apiKeysByHash:         make(map[string]*models.APIKey),
		subscriptionsByUserID: make(map[int64]*models.UserSubscription),
	}

	// Check if migration from legacy db.json is required
	legacyDBFile := filepath.Join(dataDir, "db.json")
	if _, err := os.Stat(legacyDBFile); err == nil {
		if _, uErr := os.Stat(repo.usersFile); os.IsNotExist(uErr) {
			if err := repo.migrateFromLegacyDB(legacyDBFile); err != nil {
				return nil, fmt.Errorf("migration from db.json failed: %w", err)
			}
		}
	}

	// Load domain storage files
	if err := repo.loadUsers(); err != nil {
		return nil, fmt.Errorf("failed loading users storage: %w", err)
	}
	if err := repo.loadSessions(); err != nil {
		return nil, fmt.Errorf("failed loading sessions storage: %w", err)
	}
	if err := repo.loadMenu(); err != nil {
		return nil, fmt.Errorf("failed loading menu storage: %w", err)
	}
	if err := repo.loadOrders(); err != nil {
		return nil, fmt.Errorf("failed loading orders storage: %w", err)
	}
	if err := repo.loadAuditLogs(); err != nil {
		return nil, fmt.Errorf("failed loading audit logs storage: %w", err)
	}
	if err := repo.loadCoupons(); err != nil {
		return nil, fmt.Errorf("failed loading coupons storage: %w", err)
	}
	if err := repo.loadInventory(); err != nil {
		return nil, fmt.Errorf("failed loading inventory storage: %w", err)
	}
	if err := repo.loadExpenses(); err != nil {
		return nil, fmt.Errorf("failed loading expenses storage: %w", err)
	}
	if err := repo.loadAPIKeys(); err != nil {
		return nil, fmt.Errorf("failed loading API keys storage: %w", err)
	}
	if err := repo.loadMemberships(); err != nil {
		return nil, fmt.Errorf("failed loading memberships storage: %w", err)
	}
	if err := repo.loadExpenses(); err != nil {
		return nil, fmt.Errorf("failed loading expenses storage: %w", err)
	}
	if err := repo.loadAPIKeys(); err != nil {
		return nil, fmt.Errorf("failed loading API keys storage: %w", err)
	}
	if err := repo.loadExpenses(); err != nil {
		return nil, fmt.Errorf("failed loading expenses storage: %w", err)
	}

	return repo, nil
}

// Atomic file save helper (write to .tmp then rename)
func saveAtomic(filePath string, v interface{}) error {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, bytes, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, filePath)
}

// Migration from legacy monolithic db.json to domain files
func (r *MultiFileRepository) migrateFromLegacyDB(legacyPath string) error {
	bytes, err := os.ReadFile(legacyPath)
	if err != nil {
		return err
	}
	var legacy struct {
		NextUserID          int64             `json:"next_user_id"`
		NextMenuItemID      int64             `json:"next_menu_item_id"`
		NextOrderID         int64             `json:"next_order_id"`
		NextAuditLogID      int64             `json:"next_audit_log_id"`
		Users               []models.User     `json:"users"`
		Sessions            []models.Session  `json:"sessions"`
		MenuItems           []models.MenuItem `json:"menu_items"`
		Orders              []models.Order    `json:"orders"`
		AuditLogs           []models.AuditLog `json:"audit_logs"`
		CancellationReasons []string          `json:"cancellation_reasons"`
	}
	if err := json.Unmarshal(bytes, &legacy); err != nil {
		return err
	}

	_ = saveAtomic(r.usersFile, usersSchema{NextUserID: legacy.NextUserID, Users: legacy.Users})
	_ = saveAtomic(r.sessionsFile, sessionsSchema{Sessions: legacy.Sessions})
	_ = saveAtomic(r.menuFile, menuSchema{NextMenuItemID: legacy.NextMenuItemID, MenuItems: legacy.MenuItems})
	_ = saveAtomic(r.ordersFile, ordersSchema{NextOrderID: legacy.NextOrderID, CancellationReasons: legacy.CancellationReasons, Orders: legacy.Orders})
	_ = saveAtomic(r.auditLogsFile, auditLogsSchema{NextAuditLogID: legacy.NextAuditLogID, AuditLogs: legacy.AuditLogs})
	return nil
}

// --- Users Storage Operations ---

func (r *MultiFileRepository) loadUsers() error {
	r.usersMu.Lock()
	defer r.usersMu.Unlock()

	if _, err := os.Stat(r.usersFile); os.IsNotExist(err) {
		r.seedDefaultUsers()
		return r.saveUsersLocked()
	}

	bytes, err := os.ReadFile(r.usersFile)
	if err != nil {
		return err
	}
	var s usersSchema
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	r.users = s.Users
	r.nextUserID.Store(s.NextUserID)
	if r.nextUserID.Load() == 0 {
		r.nextUserID.Store(1)
	}

	// Rebuild O(1) in-memory maps
	r.usersByEmail = make(map[string]*models.User)
	r.usersByID = make(map[int64]*models.User)
	for i := range r.users {
		u := &r.users[i]
		r.usersByEmail[strings.ToLower(u.Email)] = u
		r.usersByID[u.ID] = u
	}
	return nil
}

func (r *MultiFileRepository) seedDefaultUsers() {
	superAdminSalt := GenerateSalt()
	adminSalt := GenerateSalt()
	clientSalt := GenerateSalt()
	staffSalt := GenerateSalt()

	r.users = []models.User{
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
			Name:         "TeaChar Manager",
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
			MobileNumber: "9988776655",
			PasswordHash: HashPassword("Staff@123", staffSalt),
			Salt:         staffSalt,
			Role:         "staff",
			CreatedAt:    time.Now(),
		},
	}
	r.nextUserID.Store(5)
	r.usersByEmail = make(map[string]*models.User)
	r.usersByID = make(map[int64]*models.User)
	for i := range r.users {
		u := &r.users[i]
		r.usersByEmail[strings.ToLower(u.Email)] = u
		r.usersByID[u.ID] = u
	}
}

func (r *MultiFileRepository) saveUsersLocked() error {
	s := usersSchema{
		NextUserID: r.nextUserID.Load(),
		Users:      r.users,
	}
	return saveAtomic(r.usersFile, s)
}

func (r *MultiFileRepository) CreateUser(ctx context.Context, user models.User) (*models.User, error) {
	r.usersMu.Lock()
	defer r.usersMu.Unlock()

	emailLower := strings.ToLower(user.Email)
	if _, exists := r.usersByEmail[emailLower]; exists {
		return nil, errors.New("user with this email already exists")
	}

	user.ID = r.nextUserID.Add(1) - 1
	if user.Role == "" {
		user.Role = "client"
	}
	user.CreatedAt = time.Now()

	r.users = append(r.users, user)
	savedUser := &r.users[len(r.users)-1]
	r.usersByEmail[emailLower] = savedUser
	r.usersByID[user.ID] = savedUser

	if err := r.saveUsersLocked(); err != nil {
		return nil, err
	}
	return savedUser, nil
}

func (r *MultiFileRepository) UpdateUser(ctx context.Context, user models.User) (*models.User, error) {
	r.usersMu.Lock()
	defer r.usersMu.Unlock()

	for i, u := range r.users {
		if u.ID == user.ID {
			// Update email map if email changed
			oldEmailLower := strings.ToLower(u.Email)
			newEmailLower := strings.ToLower(user.Email)

			if oldEmailLower != newEmailLower {
				if _, exists := r.usersByEmail[newEmailLower]; exists {
					return nil, errors.New("user with this email already exists")
				}
				delete(r.usersByEmail, oldEmailLower)
			}

			r.users[i] = user
			savedUser := &r.users[i]
			r.usersByEmail[newEmailLower] = savedUser
			r.usersByID[user.ID] = savedUser

			if err := r.saveUsersLocked(); err != nil {
				return nil, err
			}
			return savedUser, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *MultiFileRepository) DeleteUser(ctx context.Context, id int64) error {
	r.usersMu.Lock()
	defer r.usersMu.Unlock()

	for i, u := range r.users {
		if u.ID == id {
			delete(r.usersByEmail, strings.ToLower(u.Email))
			delete(r.usersByID, u.ID)
			r.users = append(r.users[:i], r.users[i+1:]...)
			return r.saveUsersLocked()
		}
	}
	return errors.New("user not found")
}

func (r *MultiFileRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	r.usersMu.RLock()
	defer r.usersMu.RUnlock()

	u, exists := r.usersByEmail[strings.ToLower(email)]
	if !exists {
		return nil, errors.New("user not found")
	}
	cpy := *u
	return &cpy, nil
}

func (r *MultiFileRepository) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	r.usersMu.RLock()
	defer r.usersMu.RUnlock()

	u, exists := r.usersByID[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	cpy := *u
	return &cpy, nil
}

func (r *MultiFileRepository) GetAllUsers(ctx context.Context) ([]models.User, error) {
	r.usersMu.RLock()
	defer r.usersMu.RUnlock()

	result := make([]models.User, len(r.users))
	copy(result, r.users)
	return result, nil
}

// --- Sessions Storage Operations ---

func (r *MultiFileRepository) loadSessions() error {
	r.sessionsMu.Lock()
	defer r.sessionsMu.Unlock()

	if _, err := os.Stat(r.sessionsFile); os.IsNotExist(err) {
		r.sessions = []models.Session{}
		return r.saveSessionsLocked()
	}

	bytes, err := os.ReadFile(r.sessionsFile)
	if err != nil {
		return err
	}
	var s sessionsSchema
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	r.sessions = s.Sessions
	r.sessionsByToken = make(map[string]*models.Session)
	for i := range r.sessions {
		sess := &r.sessions[i]
		r.sessionsByToken[sess.ID] = sess
	}
	return nil
}

func (r *MultiFileRepository) saveSessionsLocked() error {
	s := sessionsSchema{Sessions: r.sessions}
	return saveAtomic(r.sessionsFile, s)
}

func (r *MultiFileRepository) CreateSession(ctx context.Context, session models.Session) error {
	r.sessionsMu.Lock()
	defer r.sessionsMu.Unlock()

	r.sessions = append(r.sessions, session)
	r.sessionsByToken[session.ID] = &r.sessions[len(r.sessions)-1]
	return r.saveSessionsLocked()
}

func (r *MultiFileRepository) GetSession(ctx context.Context, token string) (*models.Session, error) {
	r.sessionsMu.RLock()
	defer r.sessionsMu.RUnlock()

	sess, exists := r.sessionsByToken[token]
	if !exists {
		return nil, errors.New("session not found")
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, errors.New("session expired")
	}
	cpy := *sess
	return &cpy, nil
}

func (r *MultiFileRepository) DeleteSession(ctx context.Context, token string) error {
	r.sessionsMu.Lock()
	defer r.sessionsMu.Unlock()

	delete(r.sessionsByToken, token)
	var updated []models.Session
	for _, s := range r.sessions {
		if s.ID != token {
			updated = append(updated, s)
		}
	}
	r.sessions = updated
	return r.saveSessionsLocked()
}

// --- Menu Storage Operations ---

func (r *MultiFileRepository) loadMenu() error {
	r.menuMu.Lock()
	defer r.menuMu.Unlock()

	if _, err := os.Stat(r.menuFile); os.IsNotExist(err) {
		r.seedDefaultMenu()
		return r.saveMenuLocked()
	}

	bytes, err := os.ReadFile(r.menuFile)
	if err != nil {
		return err
	}
	var s menuSchema
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	r.menuItems = s.MenuItems
	r.nextMenuItemID.Store(s.NextMenuItemID)
	if r.nextMenuItemID.Load() == 0 {
		r.nextMenuItemID.Store(1)
	}

	r.menuItemsByID = make(map[int64]*models.MenuItem)
	for i := range r.menuItems {
		item := &r.menuItems[i]
		r.menuItemsByID[item.ID] = item
	}
	return nil
}

func (r *MultiFileRepository) seedDefaultMenu() {
	r.menuItems = GetDefaultMenuItems()
	r.nextMenuItemID.Store(int64(len(r.menuItems) + 1))
	r.menuItemsByID = make(map[int64]*models.MenuItem)
	for i := range r.menuItems {
		item := &r.menuItems[i]
		r.menuItemsByID[item.ID] = item
	}
}

func GetDefaultMenuItems() []models.MenuItem {
	return []models.MenuItem{
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
	}
}

func (r *MultiFileRepository) saveMenuLocked() error {
	s := menuSchema{
		NextMenuItemID: r.nextMenuItemID.Load(),
		MenuItems:      r.menuItems,
	}
	return saveAtomic(r.menuFile, s)
}

func (r *MultiFileRepository) GetAll(ctx context.Context) (map[string][]models.MenuItem, error) {
	r.menuMu.RLock()
	defer r.menuMu.RUnlock()

	categorized := make(map[string][]models.MenuItem)
	for _, item := range r.menuItems {
		categorized[item.Category] = append(categorized[item.Category], item)
	}
	return categorized, nil
}

func (r *MultiFileRepository) GetByID(ctx context.Context, id int64) (*models.MenuItem, error) {
	r.menuMu.RLock()
	defer r.menuMu.RUnlock()

	item, exists := r.menuItemsByID[id]
	if !exists {
		return nil, errors.New("menu item not found")
	}
	cpy := *item
	return &cpy, nil
}

func (r *MultiFileRepository) GetFeatured(ctx context.Context) ([]models.MenuItem, error) {
	r.menuMu.RLock()
	defer r.menuMu.RUnlock()

	var featured []models.MenuItem
	for _, item := range r.menuItems {
		if item.Available {
			featured = append(featured, item)
		}
	}
	return featured, nil
}

func (r *MultiFileRepository) Create(ctx context.Context, item models.MenuItem) (*models.MenuItem, error) {
	r.menuMu.Lock()
	defer r.menuMu.Unlock()

	item.ID = r.nextMenuItemID.Add(1) - 1
	item.Available = true

	r.menuItems = append(r.menuItems, item)
	saved := &r.menuItems[len(r.menuItems)-1]
	r.menuItemsByID[item.ID] = saved

	if err := r.saveMenuLocked(); err != nil {
		return nil, err
	}
	return saved, nil
}

func (r *MultiFileRepository) Update(ctx context.Context, item models.MenuItem) error {
	r.menuMu.Lock()
	defer r.menuMu.Unlock()

	target, exists := r.menuItemsByID[item.ID]
	if !exists {
		return errors.New("menu item not found")
	}

	target.Name = item.Name
	target.Description = item.Description
	target.Price = item.Price
	target.Category = item.Category
	target.Image = item.Image
	target.Available = item.Available

	return r.saveMenuLocked()
}

func (r *MultiFileRepository) Delete(ctx context.Context, id int64) error {
	r.menuMu.Lock()
	defer r.menuMu.Unlock()

	if _, exists := r.menuItemsByID[id]; !exists {
		return errors.New("menu item not found")
	}

	delete(r.menuItemsByID, id)
	var updated []models.MenuItem
	for _, item := range r.menuItems {
		if item.ID != id {
			updated = append(updated, item)
		}
	}
	r.menuItems = updated
	return r.saveMenuLocked()
}

func (r *MultiFileRepository) ToggleAvailability(ctx context.Context, id int64) error {
	r.menuMu.Lock()
	defer r.menuMu.Unlock()

	target, exists := r.menuItemsByID[id]
	if !exists {
		return errors.New("menu item not found")
	}
	target.Available = !target.Available
	return r.saveMenuLocked()
}

// --- Orders Storage Operations ---

func (r *MultiFileRepository) loadOrders() error {
	r.ordersMu.Lock()
	defer r.ordersMu.Unlock()

	if _, err := os.Stat(r.ordersFile); os.IsNotExist(err) {
		r.cancellationReasons = []string{
			"Item out of stock",
			"Kitchen load exceeded / High volume",
			"Customer requested cancellation",
			"Raw ingredients unavailable",
			"End of operational hours / Store closing",
			"Delivery location unreachable",
		}
		r.orders = []models.Order{}
		r.nextOrderID.Store(1)
		return r.saveOrdersLocked()
	}

	bytes, err := os.ReadFile(r.ordersFile)
	if err != nil {
		return err
	}
	var s ordersSchema
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	r.orders = s.Orders
	r.cancellationReasons = s.CancellationReasons
	if len(r.cancellationReasons) == 0 {
		r.cancellationReasons = []string{
			"Item out of stock",
			"Kitchen load exceeded / High volume",
			"Customer requested cancellation",
			"Raw ingredients unavailable",
			"End of operational hours / Store closing",
			"Delivery location unreachable",
		}
	}
	r.nextOrderID.Store(s.NextOrderID)
	if r.nextOrderID.Load() == 0 {
		r.nextOrderID.Store(1)
	}

	r.ordersByID = make(map[int64]*models.Order)
	for i := range r.orders {
		o := &r.orders[i]
		r.ordersByID[o.ID] = o
	}
	return nil
}

func (r *MultiFileRepository) saveOrdersLocked() error {
	s := ordersSchema{
		NextOrderID:         r.nextOrderID.Load(),
		CancellationReasons: r.cancellationReasons,
		Orders:              r.orders,
	}
	return saveAtomic(r.ordersFile, s)
}

func (r *MultiFileRepository) CreateOrder(ctx context.Context, order models.Order) (*models.Order, error) {
	r.ordersMu.Lock()
	defer r.ordersMu.Unlock()

	order.ID = r.nextOrderID.Add(1) - 1
	order.CreatedAt = time.Now()
	order.Status = "Pending"

	var subtotal float64
	for i, item := range order.Items {
		item.ID = int64(i + 1)
		item.OrderID = order.ID

		if menuItem, exists := r.menuItemsByID[item.MenuItemID]; exists {
			if item.ItemName == "" {
				item.ItemName = menuItem.Name
			}
			if item.Price == 0 {
				item.Price = menuItem.Price
			}
		}
		subtotal += item.Price * float64(item.Quantity)
		order.Items[i] = item
	}

	if order.SubtotalPrice == 0 {
		order.SubtotalPrice = subtotal
	}

	discountedSubtotal := order.SubtotalPrice - order.DiscountAmount
	if discountedSubtotal < 0 {
		discountedSubtotal = 0
	}

	if order.TotalPrice == 0 {
		order.TotalPrice = mathRound(discountedSubtotal * 1.05)
	}

	r.orders = append(r.orders, order)
	saved := &r.orders[len(r.orders)-1]
	r.ordersByID[order.ID] = saved

	if err := r.saveOrdersLocked(); err != nil {
		return nil, err
	}
	return saved, nil
}

func mathRound(val float64) float64 {
	return float64(int64(val*100+0.5)) / 100
}

func (r *MultiFileRepository) GetOrdersByUserID(ctx context.Context, userID int64) ([]models.Order, error) {
	r.ordersMu.RLock()
	defer r.ordersMu.RUnlock()

	var result []models.Order
	for i := len(r.orders) - 1; i >= 0; i-- {
		if r.orders[i].UserID == userID {
			result = append(result, r.orders[i])
		}
	}
	return result, nil
}

func (r *MultiFileRepository) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	r.ordersMu.RLock()
	defer r.ordersMu.RUnlock()

	var result []models.Order
	for i := len(r.orders) - 1; i >= 0; i-- {
		result = append(result, r.orders[i])
	}
	return result, nil
}

func (r *MultiFileRepository) GetOrderByID(ctx context.Context, id int64) (*models.Order, error) {
	r.ordersMu.RLock()
	defer r.ordersMu.RUnlock()

	o, exists := r.ordersByID[id]
	if !exists {
		return nil, errors.New("order not found")
	}
	cpy := *o
	return &cpy, nil
}

func (r *MultiFileRepository) UpdateOrderStatus(ctx context.Context, id int64, status string) error {
	return r.UpdateOrderStatusWithStaff(ctx, id, status, 0, "", "")
}

func (r *MultiFileRepository) UpdateOrderStatusWithStaff(ctx context.Context, id int64, status string, staffID int64, staffName string, cancellationReason string) error {
	r.ordersMu.Lock()
	defer r.ordersMu.Unlock()

	order, exists := r.ordersByID[id]
	if !exists {
		return errors.New("order not found")
	}

	order.Status = status
	if staffID != 0 {
		order.AssignedStaffID = staffID
		order.AssignedStaffName = staffName
		if order.AssignedBy == "" {
			order.AssignedBy = "Self Claimed"
		}
		if order.AssignedAt == nil {
			now := time.Now()
			order.AssignedAt = &now
		}
	}
	if cancellationReason != "" {
		order.CancellationReason = cancellationReason
	}
	if status == "Completed" && order.CompletedAt == nil {
		now := time.Now()
		order.CompletedAt = &now
		duration := int(now.Sub(order.CreatedAt).Minutes())
		if duration < 1 {
			duration = 1
		}
		order.FulfillmentMinutes = duration
	}

	return r.saveOrdersLocked()
}

func (r *MultiFileRepository) AssignOrderToStaff(ctx context.Context, id int64, staffID int64, staffName string, assignedBy string, estimatedMinutes int) error {
	r.ordersMu.Lock()
	defer r.ordersMu.Unlock()

	order, exists := r.ordersByID[id]
	if !exists {
		return errors.New("order not found")
	}

	now := time.Now()
	order.AssignedStaffID = staffID
	order.AssignedStaffName = staffName
	order.AssignedBy = assignedBy
	order.AssignedAt = &now
	if estimatedMinutes > 0 {
		order.EstimatedMinutes = estimatedMinutes
	} else if order.EstimatedMinutes <= 0 {
		order.EstimatedMinutes = 20
	}

	return r.saveOrdersLocked()
}

func (r *MultiFileRepository) GetCancellationReasons(ctx context.Context) ([]string, error) {
	r.ordersMu.RLock()
	defer r.ordersMu.RUnlock()

	result := make([]string, len(r.cancellationReasons))
	copy(result, r.cancellationReasons)
	return result, nil
}

func (r *MultiFileRepository) AddCancellationReason(ctx context.Context, reason string) error {
	if reason == "" {
		return errors.New("cancellation reason cannot be empty")
	}

	r.ordersMu.Lock()
	defer r.ordersMu.Unlock()

	for _, existing := range r.cancellationReasons {
		if existing == reason {
			return nil
		}
	}

	r.cancellationReasons = append(r.cancellationReasons, reason)
	return r.saveOrdersLocked()
}

func (r *MultiFileRepository) DeleteCancellationReason(ctx context.Context, reason string) error {
	r.ordersMu.Lock()
	defer r.ordersMu.Unlock()

	var updated []string
	for _, existing := range r.cancellationReasons {
		if existing != reason {
			updated = append(updated, existing)
		}
	}
	r.cancellationReasons = updated
	return r.saveOrdersLocked()
}

func (r *MultiFileRepository) SaveOrderReview(ctx context.Context, id int64, rating int, review string) error {
	r.ordersMu.Lock()
	defer r.ordersMu.Unlock()

	order, exists := r.ordersByID[id]
	if !exists {
		return errors.New("order not found")
	}

	order.Rating = rating
	order.Review = review
	return r.saveOrdersLocked()
}

// --- Audit Storage Operations ---

func (r *MultiFileRepository) loadAuditLogs() error {
	r.auditLogsMu.Lock()
	defer r.auditLogsMu.Unlock()

	if _, err := os.Stat(r.auditLogsFile); os.IsNotExist(err) {
		r.auditLogs = []models.AuditLog{}
		r.nextAuditLogID.Store(1)
		return r.saveAuditLogsLocked()
	}

	bytes, err := os.ReadFile(r.auditLogsFile)
	if err != nil {
		return err
	}
	var s auditLogsSchema
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	r.auditLogs = s.AuditLogs
	r.nextAuditLogID.Store(s.NextAuditLogID)
	if r.nextAuditLogID.Load() == 0 {
		r.nextAuditLogID.Store(1)
	}
	return nil
}

func (r *MultiFileRepository) saveAuditLogsLocked() error {
	s := auditLogsSchema{
		NextAuditLogID: r.nextAuditLogID.Load(),
		AuditLogs:      r.auditLogs,
	}
	return saveAtomic(r.auditLogsFile, s)
}

func (r *MultiFileRepository) CreateAuditLog(ctx context.Context, log models.AuditLog) (*models.AuditLog, error) {
	r.auditLogsMu.Lock()
	defer r.auditLogsMu.Unlock()

	log.ID = r.nextAuditLogID.Add(1) - 1
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	r.auditLogs = append(r.auditLogs, log)
	saved := &r.auditLogs[len(r.auditLogs)-1]

	if err := r.saveAuditLogsLocked(); err != nil {
		return nil, err
	}
	return saved, nil
}

func (r *MultiFileRepository) GetAllAuditLogs(ctx context.Context) ([]models.AuditLog, error) {
	r.auditLogsMu.RLock()
	defer r.auditLogsMu.RUnlock()

	var result []models.AuditLog
	for i := len(r.auditLogs) - 1; i >= 0; i-- {
		result = append(result, r.auditLogs[i])
	}
	return result, nil
}

// --- Coupons Storage Operations ---

func (r *MultiFileRepository) loadCoupons() error {
	r.couponsMu.Lock()
	defer r.couponsMu.Unlock()

	if _, err := os.Stat(r.couponsFile); os.IsNotExist(err) {
		r.coupons = []models.Coupon{}
		r.nextCouponID.Store(1)
		return r.saveCouponsLocked()
	}

	bytes, err := os.ReadFile(r.couponsFile)
	if err != nil {
		return err
	}
	var s couponsSchema
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	r.coupons = s.Coupons
	r.nextCouponID.Store(s.NextCouponID)
	if r.nextCouponID.Load() == 0 {
		r.nextCouponID.Store(1)
	}

	r.couponsByCode = make(map[string]*models.Coupon)
	for i := range r.coupons {
		c := &r.coupons[i]
		r.couponsByCode[strings.ToUpper(c.Code)] = c
	}
	return nil
}

func (r *MultiFileRepository) saveCouponsLocked() error {
	s := couponsSchema{
		NextCouponID: r.nextCouponID.Load(),
		Coupons:      r.coupons,
	}
	return saveAtomic(r.couponsFile, s)
}

func (r *MultiFileRepository) CreateCoupon(ctx context.Context, coupon models.Coupon) (*models.Coupon, error) {
	r.couponsMu.Lock()
	defer r.couponsMu.Unlock()

	codeUpper := strings.ToUpper(strings.TrimSpace(coupon.Code))
	if codeUpper == "" {
		return nil, errors.New("coupon code cannot be empty")
	}
	if _, exists := r.couponsByCode[codeUpper]; exists {
		return nil, fmt.Errorf("coupon code '%s' already exists", codeUpper)
	}

	coupon.ID = r.nextCouponID.Add(1) - 1
	coupon.Code = codeUpper
	coupon.CreatedAt = time.Now()
	coupon.IsUsed = false

	r.coupons = append(r.coupons, coupon)
	saved := &r.coupons[len(r.coupons)-1]
	r.couponsByCode[codeUpper] = saved

	if err := r.saveCouponsLocked(); err != nil {
		return nil, err
	}
	return saved, nil
}

func (r *MultiFileRepository) GetCouponByCode(ctx context.Context, code string) (*models.Coupon, error) {
	r.couponsMu.RLock()
	defer r.couponsMu.RUnlock()

	codeUpper := strings.ToUpper(strings.TrimSpace(code))
	coupon, exists := r.couponsByCode[codeUpper]
	if !exists {
		return nil, errors.New("coupon not found")
	}
	cpy := *coupon
	return &cpy, nil
}

func (r *MultiFileRepository) GetAllCoupons(ctx context.Context) ([]models.Coupon, error) {
	r.couponsMu.RLock()
	defer r.couponsMu.RUnlock()

	result := make([]models.Coupon, len(r.coupons))
	copy(result, r.coupons)
	return result, nil
}

func (r *MultiFileRepository) MarkCouponUsed(ctx context.Context, code string, orderID int64) error {
	r.couponsMu.Lock()
	defer r.couponsMu.Unlock()

	codeUpper := strings.ToUpper(strings.TrimSpace(code))
	coupon, exists := r.couponsByCode[codeUpper]
	if !exists {
		return errors.New("coupon not found")
	}

	if coupon.IsUsed {
		return errors.New("coupon has already been used")
	}

	now := time.Now()
	coupon.IsUsed = true
	coupon.UsedAt = &now
	coupon.UsedByOrderID = orderID

	return r.saveCouponsLocked()
}

func (r *MultiFileRepository) DeleteCoupon(ctx context.Context, id int64) error {
	r.couponsMu.Lock()
	defer r.couponsMu.Unlock()

	var targetCode string
	for _, c := range r.coupons {
		if c.ID == id {
			targetCode = strings.ToUpper(c.Code)
			break
		}
	}
	if targetCode == "" {
		return errors.New("coupon not found")
	}

	delete(r.couponsByCode, targetCode)
	var updated []models.Coupon
	for _, c := range r.coupons {
		if c.ID != id {
			updated = append(updated, c)
		}
	}
	r.coupons = updated
	return r.saveCouponsLocked()
}

// Inventory & Expense domain persistence methods
func (r *MultiFileRepository) loadInventory() error {
	r.inventoryMu.Lock()
	defer r.inventoryMu.Unlock()

	if _, err := os.Stat(r.inventoryFile); os.IsNotExist(err) {
		r.inventoryItems = []models.InventoryItem{}
		r.nextInventoryID.Store(1)
		return r.saveInventoryLocked()
	}

	bytes, err := os.ReadFile(r.inventoryFile)
	if err != nil {
		return err
	}

	var schema inventorySchema
	if err := json.Unmarshal(bytes, &schema); err != nil {
		return err
	}

	r.inventoryItems = schema.InventoryItems
	r.inventoryItemsByID = make(map[int64]*models.InventoryItem)
	var maxID int64
	for i := range r.inventoryItems {
		item := &r.inventoryItems[i]
		r.inventoryItemsByID[item.ID] = item
		if item.ID > maxID {
			maxID = item.ID
		}
	}
	if schema.NextInventoryID > maxID {
		maxID = schema.NextInventoryID - 1
	}
	r.nextInventoryID.Store(maxID + 1)
	return nil
}

func (r *MultiFileRepository) saveInventoryLocked() error {
	schema := inventorySchema{
		NextInventoryID: r.nextInventoryID.Load(),
		InventoryItems:  r.inventoryItems,
	}
	return saveAtomic(r.inventoryFile, schema)
}

func (r *MultiFileRepository) loadExpenses() error {
	r.expensesMu.Lock()
	defer r.expensesMu.Unlock()

	if _, err := os.Stat(r.expensesFile); os.IsNotExist(err) {
		r.expenses = []models.ExpenseEntry{}
		r.nextExpenseID.Store(1)
		return r.saveExpensesLocked()
	}

	bytes, err := os.ReadFile(r.expensesFile)
	if err != nil {
		return err
	}

	var schema expensesSchema
	if err := json.Unmarshal(bytes, &schema); err != nil {
		return err
	}

	r.expenses = schema.Expenses
	var maxID int64
	for _, exp := range r.expenses {
		if exp.ID > maxID {
			maxID = exp.ID
		}
	}
	if schema.NextExpenseID > maxID {
		maxID = schema.NextExpenseID - 1
	}
	r.nextExpenseID.Store(maxID + 1)
	return nil
}

func (r *MultiFileRepository) saveExpensesLocked() error {
	schema := expensesSchema{
		NextExpenseID: r.nextExpenseID.Load(),
		Expenses:      r.expenses,
	}
	return saveAtomic(r.expensesFile, schema)
}

func (r *MultiFileRepository) GetAllInventoryItems(ctx context.Context) ([]models.InventoryItem, error) {
	r.inventoryMu.RLock()
	defer r.inventoryMu.RUnlock()

	result := make([]models.InventoryItem, len(r.inventoryItems))
	copy(result, r.inventoryItems)
	return result, nil
}

func (r *MultiFileRepository) GetInventoryItemByID(ctx context.Context, id int64) (*models.InventoryItem, error) {
	r.inventoryMu.RLock()
	defer r.inventoryMu.RUnlock()

	item, exists := r.inventoryItemsByID[id]
	if !exists {
		return nil, errors.New("inventory item not found")
	}
	cpy := *item
	return &cpy, nil
}

func (r *MultiFileRepository) SaveInventoryItem(ctx context.Context, item models.InventoryItem) (*models.InventoryItem, error) {
	r.inventoryMu.Lock()
	defer r.inventoryMu.Unlock()

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
		item.ID = r.nextInventoryID.Add(1) - 1
		r.inventoryItems = append(r.inventoryItems, item)
		saved := &r.inventoryItems[len(r.inventoryItems)-1]
		r.inventoryItemsByID[item.ID] = saved
		if err := r.saveInventoryLocked(); err != nil {
			return nil, err
		}
		return saved, nil
	}

	existing, exists := r.inventoryItemsByID[item.ID]
	if !exists {
		return nil, errors.New("inventory item not found for update")
	}

	*existing = item
	if err := r.saveInventoryLocked(); err != nil {
		return nil, err
	}
	return existing, nil
}

func (r *MultiFileRepository) UpdateStockQuantity(ctx context.Context, id int64, delta float64) error {
	r.inventoryMu.Lock()
	defer r.inventoryMu.Unlock()

	item, exists := r.inventoryItemsByID[id]
	if !exists {
		return errors.New("inventory item not found")
	}

	item.StockQuantity += delta
	if item.StockQuantity < 0 {
		item.StockQuantity = 0
	}
	item.TotalValue = item.StockQuantity * item.UnitCost
	item.UpdatedAt = time.Now()

	if item.StockQuantity <= 0 {
		item.Status = "Out of Stock"
	} else if item.StockQuantity <= item.ReorderLevel {
		item.Status = "Low Stock"
	} else {
		item.Status = "In Stock"
	}

	return r.saveInventoryLocked()
}

func (r *MultiFileRepository) DeleteInventoryItem(ctx context.Context, id int64) error {
	r.inventoryMu.Lock()
	defer r.inventoryMu.Unlock()

	delete(r.inventoryItemsByID, id)
	var updated []models.InventoryItem
	for _, item := range r.inventoryItems {
		if item.ID != id {
			updated = append(updated, item)
		}
	}
	r.inventoryItems = updated
	return r.saveInventoryLocked()
}

func (r *MultiFileRepository) GetAllExpenses(ctx context.Context) ([]models.ExpenseEntry, error) {
	r.expensesMu.RLock()
	defer r.expensesMu.RUnlock()

	result := make([]models.ExpenseEntry, len(r.expenses))
	copy(result, r.expenses)
	return result, nil
}

func (r *MultiFileRepository) SaveExpense(ctx context.Context, expense models.ExpenseEntry) (*models.ExpenseEntry, error) {
	r.expensesMu.Lock()
	defer r.expensesMu.Unlock()

	now := time.Now()
	expense.CreatedAt = now
	if expense.ExpenseDate.IsZero() {
		expense.ExpenseDate = now
	}
	if expense.TotalAmount <= 0 && expense.Quantity > 0 && expense.UnitPrice > 0 {
		expense.TotalAmount = expense.Quantity * expense.UnitPrice
	}

	if expense.ID == 0 {
		expense.ID = r.nextExpenseID.Add(1) - 1
		r.expenses = append(r.expenses, expense)
		saved := &r.expenses[len(r.expenses)-1]
		if err := r.saveExpensesLocked(); err != nil {
			return nil, err
		}
		return saved, nil
	}

	for i, exp := range r.expenses {
		if exp.ID == expense.ID {
			r.expenses[i] = expense
			if err := r.saveExpensesLocked(); err != nil {
				return nil, err
			}
			return &r.expenses[i], nil
		}
	}
	return nil, errors.New("expense entry not found")
}

func (r *MultiFileRepository) DeleteExpense(ctx context.Context, id int64) error {
	r.expensesMu.Lock()
	defer r.expensesMu.Unlock()

	var updated []models.ExpenseEntry
	for _, exp := range r.expenses {
		if exp.ID != id {
			updated = append(updated, exp)
		}
	}
	r.expenses = updated
	return r.saveExpensesLocked()
}

func (r *MultiFileRepository) loadAPIKeys() error {
	r.apiKeysMu.Lock()
	defer r.apiKeysMu.Unlock()

	if _, err := os.Stat(r.apiKeysFile); os.IsNotExist(err) {
		r.apiKeys = []models.APIKey{}
		r.nextAPIKeyID.Store(1)
		return r.saveAPIKeysLocked()
	}

	data, err := os.ReadFile(r.apiKeysFile)
	if err != nil {
		return err
	}

	var schema apiKeysSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return err
	}

	r.apiKeys = schema.APIKeys
	r.nextAPIKeyID.Store(schema.NextAPIKeyID)
	if r.nextAPIKeyID.Load() <= 0 {
		r.nextAPIKeyID.Store(1)
	}

	r.apiKeysByHash = make(map[string]*models.APIKey)
	for i := range r.apiKeys {
		r.apiKeysByHash[r.apiKeys[i].KeyHash] = &r.apiKeys[i]
	}

	return nil
}

func (r *MultiFileRepository) saveAPIKeysLocked() error {
	schema := apiKeysSchema{
		NextAPIKeyID: r.nextAPIKeyID.Load(),
		APIKeys:      r.apiKeys,
	}
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := r.apiKeysFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, r.apiKeysFile)
}

func (r *MultiFileRepository) GetAllAPIKeys(ctx context.Context) ([]models.APIKey, error) {
	r.apiKeysMu.RLock()
	defer r.apiKeysMu.RUnlock()

	result := make([]models.APIKey, len(r.apiKeys))
	copy(result, r.apiKeys)
	return result, nil
}

func (r *MultiFileRepository) GetAPIKeyByID(ctx context.Context, id int64) (*models.APIKey, error) {
	r.apiKeysMu.RLock()
	defer r.apiKeysMu.RUnlock()

	for _, k := range r.apiKeys {
		if k.ID == id {
			return &k, nil
		}
	}
	return nil, errors.New("API key not found")
}

func (r *MultiFileRepository) GetAPIKeyByHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	r.apiKeysMu.RLock()
	defer r.apiKeysMu.RUnlock()

	if key, ok := r.apiKeysByHash[keyHash]; ok && key.IsActive {
		if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
			return nil, errors.New("API key expired")
		}
		return key, nil
	}
	return nil, errors.New("invalid or revoked API key")
}

func (r *MultiFileRepository) SaveAPIKey(ctx context.Context, apiKey models.APIKey) (*models.APIKey, error) {
	r.apiKeysMu.Lock()
	defer r.apiKeysMu.Unlock()

	now := time.Now()
	if apiKey.CreatedAt.IsZero() {
		apiKey.CreatedAt = now
	}

	if apiKey.ID == 0 {
		apiKey.ID = r.nextAPIKeyID.Add(1) - 1
		r.apiKeys = append(r.apiKeys, apiKey)
		saved := &r.apiKeys[len(r.apiKeys)-1]
		r.apiKeysByHash[saved.KeyHash] = saved
		if err := r.saveAPIKeysLocked(); err != nil {
			return nil, err
		}
		return saved, nil
	}

	for i, k := range r.apiKeys {
		if k.ID == apiKey.ID {
			r.apiKeys[i] = apiKey
			r.apiKeysByHash[apiKey.KeyHash] = &r.apiKeys[i]
			if err := r.saveAPIKeysLocked(); err != nil {
				return nil, err
			}
			return &r.apiKeys[i], nil
		}
	}
	return nil, errors.New("API key not found")
}

func (r *MultiFileRepository) UpdateAPIKeyLastUsed(ctx context.Context, id int64) error {
	r.apiKeysMu.Lock()
	defer r.apiKeysMu.Unlock()

	now := time.Now()
	for i, k := range r.apiKeys {
		if k.ID == id {
			r.apiKeys[i].LastUsedAt = &now
			r.apiKeysByHash[k.KeyHash] = &r.apiKeys[i]
			return r.saveAPIKeysLocked()
		}
	}
	return errors.New("API key not found")
}

func (r *MultiFileRepository) RevokeAPIKey(ctx context.Context, id int64) error {
	r.apiKeysMu.Lock()
	defer r.apiKeysMu.Unlock()

	for i, k := range r.apiKeys {
		if k.ID == id {
			r.apiKeys[i].IsActive = false
			delete(r.apiKeysByHash, k.KeyHash)
			return r.saveAPIKeysLocked()
		}
	}
	return errors.New("API key not found")
}

func (r *MultiFileRepository) loadMemberships() error {
	r.membershipsMu.Lock()
	defer r.membershipsMu.Unlock()

	if _, err := os.Stat(r.membershipsFile); os.IsNotExist(err) {
		r.subscriptions = []models.UserSubscription{}
		r.nextSubscriptionID.Store(1)
		return r.saveMembershipsLocked()
	}

	data, err := os.ReadFile(r.membershipsFile)
	if err != nil {
		return err
	}

	var schema membershipsSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return err
	}

	r.subscriptions = schema.Subscriptions
	r.nextSubscriptionID.Store(schema.NextSubscriptionID)
	if r.nextSubscriptionID.Load() <= 0 {
		r.nextSubscriptionID.Store(1)
	}

	r.subscriptionsByUserID = make(map[int64]*models.UserSubscription)
	for i := range r.subscriptions {
		if r.subscriptions[i].Status == "Active" {
			r.subscriptionsByUserID[r.subscriptions[i].UserID] = &r.subscriptions[i]
		}
	}

	return nil
}

func (r *MultiFileRepository) saveMembershipsLocked() error {
	schema := membershipsSchema{
		NextSubscriptionID: r.nextSubscriptionID.Load(),
		Subscriptions:      r.subscriptions,
	}
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := r.membershipsFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, r.membershipsFile)
}

func (r *MultiFileRepository) GetAllSubscriptions(ctx context.Context) ([]models.UserSubscription, error) {
	r.membershipsMu.RLock()
	defer r.membershipsMu.RUnlock()

	result := make([]models.UserSubscription, len(r.subscriptions))
	copy(result, r.subscriptions)
	return result, nil
}

func (r *MultiFileRepository) GetSubscriptionByUserID(ctx context.Context, userID int64) (*models.UserSubscription, error) {
	r.membershipsMu.RLock()
	defer r.membershipsMu.RUnlock()

	if sub, ok := r.subscriptionsByUserID[userID]; ok {
		if sub.Status == "Active" && time.Now().Before(sub.EndDate) {
			return sub, nil
		}
	}
	for _, sub := range r.subscriptions {
		if sub.UserID == userID && sub.Status == "Active" && time.Now().Before(sub.EndDate) {
			return &sub, nil
		}
	}
	return nil, errors.New("no active subscription found")
}

func (r *MultiFileRepository) GetSubscriptionByID(ctx context.Context, id int64) (*models.UserSubscription, error) {
	r.membershipsMu.RLock()
	defer r.membershipsMu.RUnlock()

	for _, sub := range r.subscriptions {
		if sub.ID == id {
			return &sub, nil
		}
	}
	return nil, errors.New("subscription not found")
}

func (r *MultiFileRepository) SaveSubscription(ctx context.Context, sub models.UserSubscription) (*models.UserSubscription, error) {
	r.membershipsMu.Lock()
	defer r.membershipsMu.Unlock()

	if sub.ID == 0 {
		sub.ID = r.nextSubscriptionID.Add(1) - 1
		r.subscriptions = append(r.subscriptions, sub)
		saved := &r.subscriptions[len(r.subscriptions)-1]
		if saved.Status == "Active" {
			r.subscriptionsByUserID[saved.UserID] = saved
		}
		if err := r.saveMembershipsLocked(); err != nil {
			return nil, err
		}
		return saved, nil
	}

	for i, s := range r.subscriptions {
		if s.ID == sub.ID {
			r.subscriptions[i] = sub
			if sub.Status == "Active" {
				r.subscriptionsByUserID[sub.UserID] = &r.subscriptions[i]
			}
			if err := r.saveMembershipsLocked(); err != nil {
				return nil, err
			}
			return &r.subscriptions[i], nil
		}
	}
	return nil, errors.New("subscription not found")
}

func (r *MultiFileRepository) ClaimDailyCup(ctx context.Context, subscriptionID int64) error {
	r.membershipsMu.Lock()
	defer r.membershipsMu.Unlock()

	now := time.Now()
	for i, sub := range r.subscriptions {
		if sub.ID == subscriptionID {
			// Check if new day reset applies
			if sub.LastClaimDate == nil || sub.LastClaimDate.Day() != now.Day() || sub.LastClaimDate.Month() != now.Month() {
				r.subscriptions[i].CupsClaimedToday = 0
			}

			if r.subscriptions[i].CupsClaimedToday >= sub.DailyFreeCupLimit {
				return errors.New("daily free cup limit reached for today")
			}

			r.subscriptions[i].CupsClaimedToday++
			r.subscriptions[i].LastClaimDate = &now
			if sub.Status == "Active" {
				r.subscriptionsByUserID[sub.UserID] = &r.subscriptions[i]
			}
			return r.saveMembershipsLocked()
		}
	}
	return errors.New("subscription not found")
}

func (r *MultiFileRepository) CancelSubscription(ctx context.Context, subscriptionID int64) error {
	r.membershipsMu.Lock()
	defer r.membershipsMu.Unlock()

	for i, sub := range r.subscriptions {
		if sub.ID == subscriptionID {
			r.subscriptions[i].Status = "Cancelled"
			delete(r.subscriptionsByUserID, sub.UserID)
			return r.saveMembershipsLocked()
		}
	}
	return errors.New("subscription not found")
}

func (r *MultiFileRepository) GetCafeSettings(ctx context.Context) (*models.CafeSettings, error) {
	r.settingsMu.Lock()
	defer r.settingsMu.Unlock()

	if r.cafeSettings == nil {
		var s models.CafeSettings
		if bytes, err := os.ReadFile(r.settingsFile); err == nil {
			_ = json.Unmarshal(bytes, &s)
		}
		if s.StoreName == "" {
			r.cafeSettings = &models.CafeSettings{
				StoreName:           "TEACHAR Flagship Cafe Sanctuary",
				StoreAddress:        "42 Chai Galleria, MG Road, Tech Hub District, Bangalore, 560001",
				BrewingHours:        "6:00 AM – 11:30 PM",
				StorePhone:          "+91 98765 43210",
				CurrencySymbol:      "₹",
				AnnouncementEnabled: true,
				AnnouncementText:    "Get 20% OFF your first order! Use code FRESHTEA",
				AnnouncementPhone:   "+91 98765 43210",
			}
		} else {
			r.cafeSettings = &s
		}
	}
	res := *r.cafeSettings
	return &res, nil
}

func (r *MultiFileRepository) UpdateCafeSettings(ctx context.Context, settings models.CafeSettings) error {
	r.settingsMu.Lock()
	defer r.settingsMu.Unlock()

	r.cafeSettings = &settings
	return saveAtomic(r.settingsFile, settings)
}
