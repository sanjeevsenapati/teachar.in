package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"teachar.in/models"
)

// SQLiteRepository provides SQLite persistent storage satisfying all teachar.in domain repository interfaces.
type SQLiteRepository struct {
	db     *sql.DB
	dbPath string
	mu     sync.RWMutex
}

// NewSQLiteRepository opens or creates a SQLite database at dbPath, creates schema and tables if they do not exist,
// and automatically migrates data from dataDir JSON files if the database is newly initialized.
func NewSQLiteRepository(dbPath string, dataDir string) (*SQLiteRepository, error) {
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed creating db dir %s: %w", dir, err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed opening sqlite db: %w", err)
	}

	// Optimize connection pool for SQLite
	db.SetMaxOpenConns(1) // SQLite performs best with single writer / serialized access in pure Go driver
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	repo := &SQLiteRepository{
		db:     db,
		dbPath: dbPath,
	}

	if err := repo.initSchema(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed initializing schema: %w", err)
	}

	// If the database has no users, migrate from dataDir JSON files or legacy db.json
	if err := repo.migrateIfEmpty(context.Background(), dataDir); err != nil {
		// Log warning
		fmt.Printf("Warning during SQLite auto-migration: %v\n", err)
	}

	return repo, nil
}

func (r *SQLiteRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *SQLiteRepository) initSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		mobile_number TEXT NOT NULL DEFAULT '',
		address TEXT NOT NULL DEFAULT '',
		avatar TEXT NOT NULL DEFAULT '',
		password_hash TEXT NOT NULL,
		salt TEXT NOT NULL,
		role TEXT NOT NULL,
		is_locked INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at TEXT NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

	CREATE TABLE IF NOT EXISTS menu_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL,
		price REAL NOT NULL,
		image TEXT NOT NULL DEFAULT '',
		available INTEGER NOT NULL DEFAULT 1
	);
	CREATE INDEX IF NOT EXISTS idx_menu_category ON menu_items(category);

	CREATE TABLE IF NOT EXISTS cancellation_reasons (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		reason TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL DEFAULT 0,
		customer_name TEXT NOT NULL,
		customer_phone TEXT NOT NULL DEFAULT '',
		order_type TEXT NOT NULL,
		table_number TEXT NOT NULL DEFAULT '',
		delivery_address TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'Pending',
		payment_method TEXT NOT NULL DEFAULT 'COD',
		payment_status TEXT NOT NULL DEFAULT 'Pending',
		transaction_id TEXT NOT NULL DEFAULT '',
		subtotal_price REAL NOT NULL DEFAULT 0,
		coupon_code TEXT NOT NULL DEFAULT '',
		discount_amount REAL NOT NULL DEFAULT 0,
		subscriber_discount REAL NOT NULL DEFAULT 0,
		subscriber_tier_name TEXT NOT NULL DEFAULT '',
		total_price REAL NOT NULL DEFAULT 0,
		assigned_staff_id INTEGER NOT NULL DEFAULT 0,
		assigned_staff_name TEXT NOT NULL DEFAULT '',
		assigned_by TEXT NOT NULL DEFAULT '',
		assigned_at TEXT,
		completed_at TEXT,
		estimated_minutes INTEGER NOT NULL DEFAULT 0,
		fulfillment_minutes INTEGER NOT NULL DEFAULT 0,
		cancellation_reason TEXT NOT NULL DEFAULT '',
		rating INTEGER NOT NULL DEFAULT 0,
		review TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
	CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
	CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);

	CREATE TABLE IF NOT EXISTS order_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id INTEGER NOT NULL,
		menu_item_id INTEGER NOT NULL,
		item_name TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		price REAL NOT NULL,
		FOREIGN KEY(order_id) REFERENCES orders(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL,
		actor_id INTEGER NOT NULL,
		actor_name TEXT NOT NULL,
		actor_role TEXT NOT NULL,
		action TEXT NOT NULL,
		details TEXT NOT NULL,
		ip_address TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);

	CREATE TABLE IF NOT EXISTS coupons (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		discount_type TEXT NOT NULL,
		discount_value REAL NOT NULL,
		min_order_amount REAL NOT NULL DEFAULT 0,
		expiry_date TEXT NOT NULL,
		target_user_id INTEGER NOT NULL DEFAULT 0,
		target_user_name TEXT NOT NULL DEFAULT '',
		is_used INTEGER NOT NULL DEFAULT 0,
		used_at TEXT,
		used_by_order_id INTEGER NOT NULL DEFAULT 0,
		created_by TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_coupons_code ON coupons(code);

	CREATE TABLE IF NOT EXISTS inventory_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category TEXT NOT NULL,
		item_name TEXT NOT NULL,
		unit TEXT NOT NULL,
		stock_quantity REAL NOT NULL,
		reorder_level REAL NOT NULL DEFAULT 0,
		unit_cost REAL NOT NULL DEFAULT 0,
		total_value REAL NOT NULL DEFAULT 0,
		supplier TEXT NOT NULL DEFAULT '',
		serial_number TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'In Stock',
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS expenses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category TEXT NOT NULL,
		title TEXT NOT NULL,
		supplier_vendor TEXT NOT NULL DEFAULT '',
		invoice_no TEXT NOT NULL DEFAULT '',
		quantity REAL NOT NULL DEFAULT 0,
		unit_price REAL NOT NULL DEFAULT 0,
		total_amount REAL NOT NULL,
		payment_method TEXT NOT NULL DEFAULT 'Cash',
		expense_date TEXT NOT NULL,
		notes TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS api_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		key_prefix TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'client',
		is_active INTEGER NOT NULL DEFAULT 1,
		last_used_at TEXT,
		created_at TEXT NOT NULL,
		expires_at TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);

	CREATE TABLE IF NOT EXISTS subscriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		user_name TEXT NOT NULL,
		user_email TEXT NOT NULL,
		tier_id TEXT NOT NULL,
		tier_name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'Active',
		price_paid REAL NOT NULL DEFAULT 0,
		discount_percent REAL NOT NULL DEFAULT 0,
		daily_free_cup_limit INTEGER NOT NULL DEFAULT 1,
		cups_claimed_today INTEGER NOT NULL DEFAULT 0,
		last_claim_date TEXT,
		start_date TEXT NOT NULL,
		end_date TEXT NOT NULL,
		auto_renew INTEGER NOT NULL DEFAULT 1,
		payment_method TEXT NOT NULL DEFAULT 'UPI',
		transaction_id TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);

	CREATE TABLE IF NOT EXISTS cafe_settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		store_name TEXT NOT NULL,
		store_address TEXT NOT NULL,
		brewing_hours TEXT NOT NULL,
		store_phone TEXT NOT NULL,
		currency_symbol TEXT NOT NULL,
		announcement_enabled INTEGER NOT NULL DEFAULT 1,
		announcement_text TEXT NOT NULL DEFAULT '',
		announcement_phone TEXT NOT NULL DEFAULT ''
	);
	`

	_, err := r.db.ExecContext(ctx, schema)
	return err
}

func (r *SQLiteRepository) migrateIfEmpty(ctx context.Context, dataDir string) error {
	var count int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // Already populated
	}

	// 1. Check if multi-file domain files exist in dataDir
	usersFile := filepath.Join(dataDir, "users.json")
	legacyDBFile := filepath.Join(dataDir, "db.json")

	if _, err := os.Stat(usersFile); err == nil {
		// Migrate multi-file domain JSONs
		return r.migrateFromMultiFiles(ctx, dataDir)
	} else if _, err := os.Stat(legacyDBFile); err == nil {
		// Migrate legacy db.json
		return r.migrateFromLegacyJSON(ctx, legacyDBFile)
	}

	// If neither exists, insert default seeds
	return r.seedDefaults(ctx)
}

func (r *SQLiteRepository) seedDefaults(ctx context.Context) error {
	now := time.Now().Format(time.RFC3339)
	users := []models.User{
		{
			ID: 1, Name: "Super Admin", Email: "superadmin@teachar.in", MobileNumber: "9000000000",
			PasswordHash: "7531f4ee48a71c86a706bfbb4b81b0c5786a08316f6b850cc35059e1fb676a28",
			Salt: "a1b2c3d4e5f67890a1b2c3d4e5f67890", Role: "superadmin",
		},
		{
			ID: 2, Name: "TEACHAR Admin", Email: "admin@teachar.in", MobileNumber: "9876543210",
			PasswordHash: "b00ab214e5f11a963c4b95a12e24db293fa77cdfe09afcd033e9a6aefc3495cc",
			Salt: "5c7fa96d763aee13ce9ac32f540ee8c6", Role: "admin",
		},
		{
			ID: 3, Name: "Sample Client", Email: "client@teachar.in", MobileNumber: "9123456789",
			PasswordHash: "46a94ba558b65a14649e27bfb375689dc39a0b059b86a34534b2b50380ae48f4",
			Salt: "127ad518779095d235fa168d14f2280b", Role: "client",
		},
		{
			ID: 4, Name: "Counter Staff", Email: "staff@teachar.in", MobileNumber: "9555544444",
			PasswordHash: "20eb1aa7c13b8cbfdf5e7d071c403a9c9d5eccad7d45957064898ae54cf4993f",
			Salt: "b2c3d4e5f67890a1b2c3d4e5f67890a1", Role: "staff",
		},
	}

	for _, u := range users {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO users (id, name, email, mobile_number, address, avatar, password_hash, salt, role, is_locked, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 'Active', ?)
		`, u.ID, u.Name, u.Email, u.MobileNumber, u.Address, u.Avatar, u.PasswordHash, u.Salt, u.Role, now)
		if err != nil {
			return err
		}
	}

	// Seed default menu items
	defaultMenu := []models.MenuItem{
		{ID: 1, Name: "Masala Tea", Description: "A classic blend of black tea and aromatic spices.", Category: "Tea", Price: 30, Image: "/static/images/masala-tea.jpg", Available: true},
		{ID: 2, Name: "Ginger Tea", Description: "Zesty and refreshing tea with a ginger kick.", Category: "Tea", Price: 30, Image: "/static/images/ginger-tea.jpg", Available: true},
		{ID: 3, Name: "Lemon Tea", Description: "Light and tangy, perfect for a fresh start.", Category: "Tea", Price: 30, Image: "/static/images/lemon-tea.jpg", Available: true},
		{ID: 4, Name: "Cold Coffee", Description: "Rich, creamy, and perfectly chilled.", Category: "Coffee", Price: 80, Image: "/static/images/cold-coffee.jpg", Available: true},
		{ID: 5, Name: "Hot Coffee", Description: "A strong and aromatic freshly brewed coffee.", Category: "Coffee", Price: 40, Image: "/static/images/hot-coffee.jpg", Available: true},
		{ID: 6, Name: "Veg Sandwich", Description: "A wholesome sandwich with fresh vegetables.", Category: "Snacks", Price: 70, Image: "/static/images/veg-sandwich.jpg", Available: true},
		{ID: 7, Name: "Samosa", Description: "Crispy pastry filled with spiced potatoes.", Category: "Snacks", Price: 25, Image: "/static/images/samosa.jpg", Available: true},
		{ID: 8, Name: "Coke", Description: "Chilled Coca-Cola.", Category: "Cold Drinks", Price: 40, Image: "/static/images/coke.jpg", Available: true},
	}

	for _, item := range defaultMenu {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO menu_items (id, name, description, category, price, image, available)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, item.ID, item.Name, item.Description, item.Category, item.Price, item.Image, boolToInt(item.Available))
		if err != nil {
			return err
		}
	}

	// Seed Cancellation Reasons
	reasons := []string{
		"Item out of stock",
		"Kitchen load exceeded / High volume",
		"Customer requested cancellation",
		"End of operational hours / Store closing",
	}
	for _, reason := range reasons {
		_, _ = r.db.ExecContext(ctx, "INSERT OR IGNORE INTO cancellation_reasons (reason) VALUES (?)", reason)
	}

	// Seed Cafe Settings
	_, _ = r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO cafe_settings (id, store_name, store_address, brewing_hours, store_phone, currency_symbol, announcement_enabled, announcement_text, announcement_phone)
		VALUES (1, 'TEACHAR Flagship Cafe Sanctuary', '42 Chai Galleria, MG Road, Tech Hub District, Bangalore, 560001', '6:00 AM – 11:30 PM', '+91 98765 43210', '₹', 1, 'Get 20% OFF your first order! Use code FRESHTEA', '+91 98765 43210')
	`)

	return nil
}

func (r *SQLiteRepository) migrateFromMultiFiles(ctx context.Context, dataDir string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Users
	if usersBytes, err := os.ReadFile(filepath.Join(dataDir, "users.json")); err == nil {
		var schema struct {
			Users []models.User `json:"users"`
		}
		if err := json.Unmarshal(usersBytes, &schema); err == nil {
			for _, u := range schema.Users {
				createdAt := u.CreatedAt.Format(time.RFC3339)
				if u.CreatedAt.IsZero() {
					createdAt = time.Now().Format(time.RFC3339)
				}
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO users (id, name, email, mobile_number, address, avatar, password_hash, salt, role, is_locked, status, created_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, u.ID, u.Name, u.Email, u.MobileNumber, u.Address, u.Avatar, u.PasswordHash, u.Salt, u.Role, boolToInt(u.IsLocked), u.Status, createdAt)
			}
		}
	}

	// 2. Sessions
	if sessBytes, err := os.ReadFile(filepath.Join(dataDir, "sessions.json")); err == nil {
		var schema struct {
			Sessions []models.Session `json:"sessions"`
		}
		if err := json.Unmarshal(sessBytes, &schema); err == nil {
			for _, s := range schema.Sessions {
				expiresAt := s.ExpiresAt.Format(time.RFC3339)
				_, _ = tx.ExecContext(ctx, `
					INSERT OR REPLACE INTO sessions (id, user_id, expires_at)
					VALUES (?, ?, ?)
				`, s.ID, s.UserID, expiresAt)
			}
		}
	}

	// 3. Menu
	if menuBytes, err := os.ReadFile(filepath.Join(dataDir, "menu.json")); err == nil {
		var schema struct {
			MenuItems []models.MenuItem `json:"menu_items"`
		}
		if err := json.Unmarshal(menuBytes, &schema); err == nil {
			for _, item := range schema.MenuItems {
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO menu_items (id, name, description, category, price, image, available)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, item.ID, item.Name, item.Description, item.Category, item.Price, item.Image, boolToInt(item.Available))
			}
		}
	}

	// 4. Orders & Cancellation reasons
	if ordersBytes, err := os.ReadFile(filepath.Join(dataDir, "orders.json")); err == nil {
		var schema struct {
			CancellationReasons []string       `json:"cancellation_reasons"`
			Orders              []models.Order `json:"orders"`
		}
		if err := json.Unmarshal(ordersBytes, &schema); err == nil {
			for _, r := range schema.CancellationReasons {
				_, _ = tx.ExecContext(ctx, "INSERT OR IGNORE INTO cancellation_reasons (reason) VALUES (?)", r)
			}
			for _, o := range schema.Orders {
				createdAt := o.CreatedAt.Format(time.RFC3339)
				if o.CreatedAt.IsZero() {
					createdAt = time.Now().Format(time.RFC3339)
				}
				var assignedAtStr, completedAtStr *string
				if o.AssignedAt != nil {
					s := o.AssignedAt.Format(time.RFC3339)
					assignedAtStr = &s
				}
				if o.CompletedAt != nil {
					s := o.CompletedAt.Format(time.RFC3339)
					completedAtStr = &s
				}

				_, _ = tx.ExecContext(ctx, `
					INSERT INTO orders (
						id, user_id, customer_name, customer_phone, order_type, table_number, delivery_address,
						status, payment_method, payment_status, transaction_id, subtotal_price, coupon_code,
						discount_amount, subscriber_discount, subscriber_tier_name, total_price, assigned_staff_id,
						assigned_staff_name, assigned_by, assigned_at, completed_at, estimated_minutes,
						fulfillment_minutes, cancellation_reason, rating, review, created_at
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, o.ID, o.UserID, o.CustomerName, o.CustomerPhone, o.OrderType, o.TableNumber, o.DeliveryAddress,
					o.Status, o.PaymentMethod, o.PaymentStatus, o.TransactionID, o.SubtotalPrice, o.CouponCode,
					o.DiscountAmount, o.SubscriberDiscount, o.SubscriberTierName, o.TotalPrice, o.AssignedStaffID,
					o.AssignedStaffName, o.AssignedBy, assignedAtStr, completedAtStr, o.EstimatedMinutes,
					o.FulfillmentMinutes, o.CancellationReason, o.Rating, o.Review, createdAt)

				for _, itm := range o.Items {
					_, _ = tx.ExecContext(ctx, `
						INSERT INTO order_items (order_id, menu_item_id, item_name, quantity, price)
						VALUES (?, ?, ?, ?, ?)
					`, o.ID, itm.MenuItemID, itm.ItemName, itm.Quantity, itm.Price)
				}
			}
		}
	}

	// 5. Coupons
	if couponsBytes, err := os.ReadFile(filepath.Join(dataDir, "coupons.json")); err == nil {
		var schema struct {
			Coupons []models.Coupon `json:"coupons"`
		}
		if err := json.Unmarshal(couponsBytes, &schema); err == nil {
			for _, c := range schema.Coupons {
				createdAt := c.CreatedAt.Format(time.RFC3339)
				expiryDate := c.ExpiryDate.Format(time.RFC3339)
				var usedAtStr *string
				if c.UsedAt != nil {
					s := c.UsedAt.Format(time.RFC3339)
					usedAtStr = &s
				}
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO coupons (
						id, code, discount_type, discount_value, min_order_amount, expiry_date,
						target_user_id, target_user_name, is_used, used_at, used_by_order_id,
						created_by, created_at
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, c.ID, c.Code, c.DiscountType, c.DiscountValue, c.MinOrderAmount, expiryDate,
					c.TargetUserID, c.TargetUserName, boolToInt(c.IsUsed), usedAtStr, c.UsedByOrderID,
					c.CreatedBy, createdAt)
			}
		}
	}

	// 6. Inventory Items
	if invBytes, err := os.ReadFile(filepath.Join(dataDir, "inventory.json")); err == nil {
		var schema struct {
			InventoryItems []models.InventoryItem `json:"inventory_items"`
		}
		if err := json.Unmarshal(invBytes, &schema); err == nil {
			for _, itm := range schema.InventoryItems {
				updatedAt := itm.UpdatedAt.Format(time.RFC3339)
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO inventory_items (
						id, category, item_name, unit, stock_quantity, reorder_level,
						unit_cost, total_value, supplier, serial_number, status, updated_at
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, itm.ID, itm.Category, itm.ItemName, itm.Unit, itm.StockQuantity, itm.ReorderLevel,
					itm.UnitCost, itm.TotalValue, itm.Supplier, itm.SerialNumber, itm.Status, updatedAt)
			}
		}
	}

	// 7. Expenses
	if expBytes, err := os.ReadFile(filepath.Join(dataDir, "expenses.json")); err == nil {
		var schema struct {
			Expenses []models.ExpenseEntry `json:"expenses"`
		}
		if err := json.Unmarshal(expBytes, &schema); err == nil {
			for _, exp := range schema.Expenses {
				expDate := exp.ExpenseDate.Format(time.RFC3339)
				createdAt := exp.CreatedAt.Format(time.RFC3339)
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO expenses (
						id, category, title, supplier_vendor, invoice_no, quantity,
						unit_price, total_amount, payment_method, expense_date, notes, created_at
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, exp.ID, exp.Category, exp.Title, exp.SupplierVendor, exp.InvoiceNo, exp.Quantity,
					exp.UnitPrice, exp.TotalAmount, exp.PaymentMethod, expDate, exp.Notes, createdAt)
			}
		}
	}

	// 8. API Keys
	if keysBytes, err := os.ReadFile(filepath.Join(dataDir, "api_keys.json")); err == nil {
		var schema struct {
			APIKeys []models.APIKey `json:"api_keys"`
		}
		if err := json.Unmarshal(keysBytes, &schema); err == nil {
			for _, k := range schema.APIKeys {
				createdAt := k.CreatedAt.Format(time.RFC3339)
				var lastUsedStr, expiresStr *string
				if k.LastUsedAt != nil {
					s := k.LastUsedAt.Format(time.RFC3339)
					lastUsedStr = &s
				}
				if k.ExpiresAt != nil {
					s := k.ExpiresAt.Format(time.RFC3339)
					expiresStr = &s
				}
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO api_keys (
						id, name, key_hash, key_prefix, role, is_active, last_used_at, created_at, expires_at
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, k.ID, k.Name, k.KeyHash, k.KeyPrefix, k.Role, boolToInt(k.IsActive), lastUsedStr, createdAt, expiresStr)
			}
		}
	}

	// 9. Memberships
	if memBytes, err := os.ReadFile(filepath.Join(dataDir, "memberships.json")); err == nil {
		var schema struct {
			Subscriptions []models.UserSubscription `json:"subscriptions"`
		}
		if err := json.Unmarshal(memBytes, &schema); err == nil {
			for _, sub := range schema.Subscriptions {
				startDate := sub.StartDate.Format(time.RFC3339)
				endDate := sub.EndDate.Format(time.RFC3339)
				var lastClaimStr *string
				if sub.LastClaimDate != nil {
					s := sub.LastClaimDate.Format(time.RFC3339)
					lastClaimStr = &s
				}
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO subscriptions (
						id, user_id, user_name, user_email, tier_id, tier_name, status,
						price_paid, discount_percent, daily_free_cup_limit, cups_claimed_today,
						last_claim_date, start_date, end_date, auto_renew, payment_method, transaction_id
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, sub.ID, sub.UserID, sub.UserName, sub.UserEmail, sub.TierID, sub.TierName, sub.Status,
					sub.PricePaid, sub.DiscountPercent, sub.DailyFreeCupLimit, sub.CupsClaimedToday,
					lastClaimStr, startDate, endDate, boolToInt(sub.AutoRenew), sub.PaymentMethod, sub.TransactionID)
			}
		}
	}

	// 10. Audit Logs
	if auditBytes, err := os.ReadFile(filepath.Join(dataDir, "audit_logs.json")); err == nil {
		var schema struct {
			AuditLogs []models.AuditLog `json:"audit_logs"`
		}
		if err := json.Unmarshal(auditBytes, &schema); err == nil {
			for _, log := range schema.AuditLogs {
				ts := log.Timestamp.Format(time.RFC3339)
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO audit_logs (
						id, timestamp, actor_id, actor_name, actor_role, action, details, ip_address
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`, log.ID, ts, log.ActorID, log.ActorName, log.ActorRole, log.Action, log.Details, log.IPAddress)
			}
		}
	}

	// 11. Cafe Settings
	var s models.CafeSettings
	if setBytes, err := os.ReadFile(filepath.Join(dataDir, "settings.json")); err == nil {
		_ = json.Unmarshal(setBytes, &s)
	}
	if s.StoreName == "" {
		s = models.CafeSettings{
			StoreName:           "TEACHAR Flagship Cafe Sanctuary",
			StoreAddress:        "42 Chai Galleria, MG Road, Tech Hub District, Bangalore, 560001",
			BrewingHours:        "6:00 AM – 11:30 PM",
			StorePhone:          "+91 98765 43210",
			CurrencySymbol:      "₹",
			AnnouncementEnabled: true,
			AnnouncementText:    "Get 20% OFF your first order! Use code FRESHTEA",
			AnnouncementPhone:   "+91 98765 43210",
		}
	}
	_, _ = tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO cafe_settings (
			id, store_name, store_address, brewing_hours, store_phone, currency_symbol,
			announcement_enabled, announcement_text, announcement_phone
		) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.StoreName, s.StoreAddress, s.BrewingHours, s.StorePhone, s.CurrencySymbol,
		boolToInt(s.AnnouncementEnabled), s.AnnouncementText, s.AnnouncementPhone)

	return tx.Commit()
}

func (r *SQLiteRepository) migrateFromLegacyJSON(ctx context.Context, legacyDBFile string) error {
	data, err := os.ReadFile(legacyDBFile)
	if err != nil {
		return err
	}

	var schema struct {
		Users               []models.User             `json:"users"`
		Sessions            []models.Session          `json:"sessions"`
		MenuItems           []models.MenuItem         `json:"menu_items"`
		Orders              []models.Order            `json:"orders"`
		AuditLogs           []models.AuditLog         `json:"audit_logs"`
		Coupons             []models.Coupon           `json:"coupons"`
		InventoryItems      []models.InventoryItem    `json:"inventory_items"`
		Expenses            []models.ExpenseEntry     `json:"expenses"`
		APIKeys             []models.APIKey           `json:"api_keys"`
		Subscriptions       []models.UserSubscription `json:"subscriptions"`
		CancellationReasons []string                  `json:"cancellation_reasons"`
	}

	if err := json.Unmarshal(data, &schema); err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, u := range schema.Users {
		createdAt := u.CreatedAt.Format(time.RFC3339)
		if u.CreatedAt.IsZero() {
			createdAt = time.Now().Format(time.RFC3339)
		}
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO users (id, name, email, mobile_number, address, avatar, password_hash, salt, role, is_locked, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, u.ID, u.Name, u.Email, u.MobileNumber, u.Address, u.Avatar, u.PasswordHash, u.Salt, u.Role, boolToInt(u.IsLocked), u.Status, createdAt)
	}

	for _, item := range schema.MenuItems {
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO menu_items (id, name, description, category, price, image, available)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, item.ID, item.Name, item.Description, item.Category, item.Price, item.Image, boolToInt(item.Available))
	}

	for _, reason := range schema.CancellationReasons {
		_, _ = tx.ExecContext(ctx, "INSERT OR IGNORE INTO cancellation_reasons (reason) VALUES (?)", reason)
	}

	for _, o := range schema.Orders {
		createdAt := o.CreatedAt.Format(time.RFC3339)
		if o.CreatedAt.IsZero() {
			createdAt = time.Now().Format(time.RFC3339)
		}
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO orders (
				id, user_id, customer_name, customer_phone, order_type, table_number, delivery_address,
				status, payment_method, payment_status, transaction_id, subtotal_price, coupon_code,
				discount_amount, subscriber_discount, subscriber_tier_name, total_price, assigned_staff_id,
				assigned_staff_name, assigned_by, cancellation_reason, rating, review, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, o.ID, o.UserID, o.CustomerName, o.CustomerPhone, o.OrderType, o.TableNumber, o.DeliveryAddress,
			o.Status, o.PaymentMethod, o.PaymentStatus, o.TransactionID, o.SubtotalPrice, o.CouponCode,
			o.DiscountAmount, o.SubscriberDiscount, o.SubscriberTierName, o.TotalPrice, o.AssignedStaffID,
			o.AssignedStaffName, o.AssignedBy, o.CancellationReason, o.Rating, o.Review, createdAt)

		for _, itm := range o.Items {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO order_items (order_id, menu_item_id, item_name, quantity, price)
				VALUES (?, ?, ?, ?, ?)
			`, o.ID, itm.MenuItemID, itm.ItemName, itm.Quantity, itm.Price)
		}
	}

	// Cafe Settings
	_, _ = tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO cafe_settings (id, store_name, store_address, brewing_hours, store_phone, currency_symbol, announcement_enabled, announcement_text, announcement_phone)
		VALUES (1, 'TEACHAR Flagship Cafe Sanctuary', '42 Chai Galleria, MG Road, Tech Hub District, Bangalore, 560001', '6:00 AM – 11:30 PM', '+91 98765 43210', '₹', 1, 'Get 20% OFF your first order! Use code FRESHTEA', '+91 98765 43210')
	`)

	return tx.Commit()
}

// ------------------------------------------------------------------------------------------------
// MenuRepository Implementation
// ------------------------------------------------------------------------------------------------

func (r *SQLiteRepository) GetAll(ctx context.Context) (map[string][]models.MenuItem, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, description, category, price, image, available FROM menu_items ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	menuMap := make(map[string][]models.MenuItem)
	for rows.Next() {
		var item models.MenuItem
		var availInt int
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Category, &item.Price, &item.Image, &availInt); err != nil {
			return nil, err
		}
		item.Available = (availInt != 0)
		menuMap[item.Category] = append(menuMap[item.Category], item)
	}
	return menuMap, rows.Err()
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (*models.MenuItem, error) {
	var item models.MenuItem
	var availInt int
	err := r.db.QueryRowContext(ctx, "SELECT id, name, description, category, price, image, available FROM menu_items WHERE id = ?", id).
		Scan(&item.ID, &item.Name, &item.Description, &item.Category, &item.Price, &item.Image, &availInt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("menu item not found")
	}
	if err != nil {
		return nil, err
	}
	item.Available = (availInt != 0)
	return &item, nil
}

func (r *SQLiteRepository) GetFeatured(ctx context.Context) ([]models.MenuItem, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, description, category, price, image, available FROM menu_items WHERE available = 1 LIMIT 4")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.MenuItem
	for rows.Next() {
		var item models.MenuItem
		var availInt int
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Category, &item.Price, &item.Image, &availInt); err != nil {
			return nil, err
		}
		item.Available = (availInt != 0)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SQLiteRepository) Create(ctx context.Context, item models.MenuItem) (*models.MenuItem, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO menu_items (name, description, category, price, image, available)
		VALUES (?, ?, ?, ?, ?, ?)
	`, item.Name, item.Description, item.Category, item.Price, item.Image, boolToInt(item.Available))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	item.ID = id
	return &item, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, item models.MenuItem) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE menu_items
		SET name = ?, description = ?, category = ?, price = ?, image = ?, available = ?
		WHERE id = ?
	`, item.Name, item.Description, item.Category, item.Price, item.Image, boolToInt(item.Available), item.ID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("menu item not found")
	}
	return nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM menu_items WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("menu item not found")
	}
	return nil
}

func (r *SQLiteRepository) ToggleAvailability(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE menu_items
		SET available = CASE WHEN available = 1 THEN 0 ELSE 1 END
		WHERE id = ?
	`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("menu item not found")
	}
	return nil
}

// ------------------------------------------------------------------------------------------------
// UserRepository Implementation
// ------------------------------------------------------------------------------------------------

func (r *SQLiteRepository) CreateUser(ctx context.Context, user models.User) (*models.User, error) {
	createdAt := user.CreatedAt.Format(time.RFC3339)
	if user.CreatedAt.IsZero() {
		createdAt = time.Now().Format(time.RFC3339)
		user.CreatedAt = time.Now()
	}
	if user.Status == "" {
		user.Status = "Active"
	}

	var query string
	var res sql.Result
	var err error

	if user.ID > 0 {
		query = `
			INSERT INTO users (id, name, email, mobile_number, address, avatar, password_hash, salt, role, is_locked, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		res, err = r.db.ExecContext(ctx, query, user.ID, user.Name, user.Email, user.MobileNumber, user.Address, user.Avatar, user.PasswordHash, user.Salt, user.Role, boolToInt(user.IsLocked), user.Status, createdAt)
	} else {
		query = `
			INSERT INTO users (name, email, mobile_number, address, avatar, password_hash, salt, role, is_locked, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		res, err = r.db.ExecContext(ctx, query, user.Name, user.Email, user.MobileNumber, user.Address, user.Avatar, user.PasswordHash, user.Salt, user.Role, boolToInt(user.IsLocked), user.Status, createdAt)
	}

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			return nil, errors.New("email already registered")
		}
		return nil, err
	}

	if user.ID <= 0 {
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		user.ID = id
	}

	return &user, nil
}

func (r *SQLiteRepository) UpdateUser(ctx context.Context, user models.User) (*models.User, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET name = ?, email = ?, mobile_number = ?, address = ?, avatar = ?, password_hash = ?, salt = ?, role = ?, is_locked = ?, status = ?
		WHERE id = ?
	`, user.Name, user.Email, user.MobileNumber, user.Address, user.Avatar, user.PasswordHash, user.Salt, user.Role, boolToInt(user.IsLocked), user.Status, user.ID)
	if err != nil {
		return nil, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (r *SQLiteRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	var isLockedInt int
	var createdStr string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, email, mobile_number, address, avatar, password_hash, salt, role, is_locked, status, created_at
		FROM users WHERE LOWER(email) = LOWER(?)
	`, strings.TrimSpace(email)).Scan(
		&u.ID, &u.Name, &u.Email, &u.MobileNumber, &u.Address, &u.Avatar, &u.PasswordHash, &u.Salt, &u.Role, &isLockedInt, &u.Status, &createdStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	u.IsLocked = (isLockedInt != 0)
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	return &u, nil
}

func (r *SQLiteRepository) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	var u models.User
	var isLockedInt int
	var createdStr string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, email, mobile_number, address, avatar, password_hash, salt, role, is_locked, status, created_at
		FROM users WHERE id = ?
	`, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.MobileNumber, &u.Address, &u.Avatar, &u.PasswordHash, &u.Salt, &u.Role, &isLockedInt, &u.Status, &createdStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	u.IsLocked = (isLockedInt != 0)
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	return &u, nil
}

func (r *SQLiteRepository) GetAllUsers(ctx context.Context) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, email, mobile_number, address, avatar, password_hash, salt, role, is_locked, status, created_at
		FROM users ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var isLockedInt int
		var createdStr string
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.MobileNumber, &u.Address, &u.Avatar, &u.PasswordHash, &u.Salt, &u.Role, &isLockedInt, &u.Status, &createdStr); err != nil {
			return nil, err
		}
		u.IsLocked = (isLockedInt != 0)
		u.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *SQLiteRepository) DeleteUser(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (r *SQLiteRepository) CreateSession(ctx context.Context, session models.Session) error {
	expiresAt := session.ExpiresAt.Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO sessions (id, user_id, expires_at)
		VALUES (?, ?, ?)
	`, session.ID, session.UserID, expiresAt)
	return err
}

func (r *SQLiteRepository) GetSession(ctx context.Context, token string) (*models.Session, error) {
	var s models.Session
	var expStr string
	err := r.db.QueryRowContext(ctx, "SELECT id, user_id, expires_at FROM sessions WHERE id = ?", token).
		Scan(&s.ID, &s.UserID, &expStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("session not found")
	}
	if err != nil {
		return nil, err
	}
	s.ExpiresAt, _ = time.Parse(time.RFC3339, expStr)
	if time.Now().After(s.ExpiresAt) {
		_ = r.DeleteSession(ctx, token)
		return nil, errors.New("session expired")
	}
	return &s, nil
}

func (r *SQLiteRepository) DeleteSession(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", token)
	return err
}

// ------------------------------------------------------------------------------------------------
// OrderRepository Implementation
// ------------------------------------------------------------------------------------------------

func (r *SQLiteRepository) CreateOrder(ctx context.Context, order models.Order) (*models.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	createdAt := order.CreatedAt.Format(time.RFC3339)
	if order.CreatedAt.IsZero() {
		createdAt = time.Now().Format(time.RFC3339)
		order.CreatedAt = time.Now()
	}
	if order.Status == "" {
		order.Status = "Pending"
	}
	if order.PaymentStatus == "" {
		order.PaymentStatus = "Pending"
	}

	var assignedAtStr, completedAtStr *string
	if order.AssignedAt != nil {
		s := order.AssignedAt.Format(time.RFC3339)
		assignedAtStr = &s
	}
	if order.CompletedAt != nil {
		s := order.CompletedAt.Format(time.RFC3339)
		completedAtStr = &s
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO orders (
			user_id, customer_name, customer_phone, order_type, table_number, delivery_address,
			status, payment_method, payment_status, transaction_id, subtotal_price, coupon_code,
			discount_amount, subscriber_discount, subscriber_tier_name, total_price, assigned_staff_id,
			assigned_staff_name, assigned_by, assigned_at, completed_at, estimated_minutes,
			fulfillment_minutes, cancellation_reason, rating, review, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, order.UserID, order.CustomerName, order.CustomerPhone, order.OrderType, order.TableNumber, order.DeliveryAddress,
		order.Status, order.PaymentMethod, order.PaymentStatus, order.TransactionID, order.SubtotalPrice, order.CouponCode,
		order.DiscountAmount, order.SubscriberDiscount, order.SubscriberTierName, order.TotalPrice, order.AssignedStaffID,
		order.AssignedStaffName, order.AssignedBy, assignedAtStr, completedAtStr, order.EstimatedMinutes,
		order.FulfillmentMinutes, order.CancellationReason, order.Rating, order.Review, createdAt)
	if err != nil {
		return nil, err
	}

	orderID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	order.ID = orderID

	for i, item := range order.Items {
		itemRes, err := tx.ExecContext(ctx, `
			INSERT INTO order_items (order_id, menu_item_id, item_name, quantity, price)
			VALUES (?, ?, ?, ?, ?)
		`, orderID, item.MenuItemID, item.ItemName, item.Quantity, item.Price)
		if err != nil {
			return nil, err
		}
		itemID, _ := itemRes.LastInsertId()
		order.Items[i].ID = itemID
		order.Items[i].OrderID = orderID
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *SQLiteRepository) GetOrdersByUserID(ctx context.Context, userID int64) ([]models.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, customer_name, customer_phone, order_type, table_number, delivery_address,
		       status, payment_method, payment_status, transaction_id, subtotal_price, coupon_code,
		       discount_amount, subscriber_discount, subscriber_tier_name, total_price, assigned_staff_id,
		       assigned_staff_name, assigned_by, assigned_at, completed_at, estimated_minutes,
		       fulfillment_minutes, cancellation_reason, rating, review, created_at
		FROM orders WHERE user_id = ? ORDER BY id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}

	// Fetch items for all orders
	for i := range orders {
		items, err := r.getOrderItems(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}

	return orders, nil
}

func (r *SQLiteRepository) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, customer_name, customer_phone, order_type, table_number, delivery_address,
		       status, payment_method, payment_status, transaction_id, subtotal_price, coupon_code,
		       discount_amount, subscriber_discount, subscriber_tier_name, total_price, assigned_staff_id,
		       assigned_staff_name, assigned_by, assigned_at, completed_at, estimated_minutes,
		       fulfillment_minutes, cancellation_reason, rating, review, created_at
		FROM orders ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}

	for i := range orders {
		items, err := r.getOrderItems(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}

	return orders, nil
}

func (r *SQLiteRepository) GetOrderByID(ctx context.Context, id int64) (*models.Order, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, customer_name, customer_phone, order_type, table_number, delivery_address,
		       status, payment_method, payment_status, transaction_id, subtotal_price, coupon_code,
		       discount_amount, subscriber_discount, subscriber_tier_name, total_price, assigned_staff_id,
		       assigned_staff_name, assigned_by, assigned_at, completed_at, estimated_minutes,
		       fulfillment_minutes, cancellation_reason, rating, review, created_at
		FROM orders WHERE id = ?
	`, id)

	o, err := scanOrderRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("order not found")
	}
	if err != nil {
		return nil, err
	}

	items, err := r.getOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items

	return o, nil
}

func (r *SQLiteRepository) UpdateOrderStatus(ctx context.Context, id int64, status string) error {
	var completedAtStr *string
	var fulfillmentMinutes int

	if status == "Completed" {
		now := time.Now()
		s := now.Format(time.RFC3339)
		completedAtStr = &s

		var createdStr string
		_ = r.db.QueryRowContext(ctx, "SELECT created_at FROM orders WHERE id = ?", id).Scan(&createdStr)
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			fulfillmentMinutes = int(now.Sub(t).Minutes())
			if fulfillmentMinutes < 0 {
				fulfillmentMinutes = 0
			}
		}
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = ?,
		    completed_at = COALESCE(?, completed_at),
		    fulfillment_minutes = CASE WHEN ? > 0 THEN ? ELSE fulfillment_minutes END
		WHERE id = ?
	`, status, completedAtStr, fulfillmentMinutes, fulfillmentMinutes, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("order not found")
	}
	return nil
}

func (r *SQLiteRepository) UpdateOrderStatusWithStaff(ctx context.Context, id int64, status string, staffID int64, staffName string, cancellationReason string) error {
	var completedAtStr *string
	var fulfillmentMinutes int

	if status == "Completed" {
		now := time.Now()
		s := now.Format(time.RFC3339)
		completedAtStr = &s

		var createdStr string
		_ = r.db.QueryRowContext(ctx, "SELECT created_at FROM orders WHERE id = ?", id).Scan(&createdStr)
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			fulfillmentMinutes = int(now.Sub(t).Minutes())
			if fulfillmentMinutes < 0 {
				fulfillmentMinutes = 0
			}
		}
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = ?,
		    assigned_staff_id = CASE WHEN ? > 0 THEN ? ELSE assigned_staff_id END,
		    assigned_staff_name = CASE WHEN ? != '' THEN ? ELSE assigned_staff_name END,
		    cancellation_reason = CASE WHEN ? != '' THEN ? ELSE cancellation_reason END,
		    completed_at = COALESCE(?, completed_at),
		    fulfillment_minutes = CASE WHEN ? > 0 THEN ? ELSE fulfillment_minutes END
		WHERE id = ?
	`, status, staffID, staffID, staffName, staffName, cancellationReason, cancellationReason, completedAtStr, fulfillmentMinutes, fulfillmentMinutes, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("order not found")
	}
	return nil
}

func (r *SQLiteRepository) AssignOrderToStaff(ctx context.Context, id int64, staffID int64, staffName string, assignedBy string, estimatedMinutes int) error {
	now := time.Now().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET assigned_staff_id = ?,
		    assigned_staff_name = ?,
		    assigned_by = ?,
		    assigned_at = ?,
		    estimated_minutes = ?
		WHERE id = ?
	`, staffID, staffName, assignedBy, now, estimatedMinutes, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("order not found")
	}
	return nil
}

func (r *SQLiteRepository) GetCancellationReasons(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT reason FROM cancellation_reasons ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reasons []string
	for rows.Next() {
		var reason string
		if err := rows.Scan(&reason); err != nil {
			return nil, err
		}
		reasons = append(reasons, reason)
	}
	return reasons, rows.Err()
}

func (r *SQLiteRepository) AddCancellationReason(ctx context.Context, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("cancellation reason cannot be empty")
	}
	_, err := r.db.ExecContext(ctx, "INSERT OR IGNORE INTO cancellation_reasons (reason) VALUES (?)", reason)
	return err
}

func (r *SQLiteRepository) DeleteCancellationReason(ctx context.Context, reason string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM cancellation_reasons WHERE reason = ?", strings.TrimSpace(reason))
	return err
}

func (r *SQLiteRepository) SaveOrderReview(ctx context.Context, id int64, rating int, review string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET rating = ?, review = ?
		WHERE id = ?
	`, rating, review, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("order not found")
	}
	return nil
}

func (r *SQLiteRepository) getOrderItems(ctx context.Context, orderID int64) ([]models.OrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, menu_item_id, item_name, quantity, price
		FROM order_items WHERE order_id = ? ORDER BY id ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.MenuItemID, &item.ItemName, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanOrder(rows *sql.Rows) (*models.Order, error) {
	var o models.Order
	var assignedAtStr, completedAtStr *string
	var createdStr string

	err := rows.Scan(
		&o.ID, &o.UserID, &o.CustomerName, &o.CustomerPhone, &o.OrderType, &o.TableNumber, &o.DeliveryAddress,
		&o.Status, &o.PaymentMethod, &o.PaymentStatus, &o.TransactionID, &o.SubtotalPrice, &o.CouponCode,
		&o.DiscountAmount, &o.SubscriberDiscount, &o.SubscriberTierName, &o.TotalPrice, &o.AssignedStaffID,
		&o.AssignedStaffName, &o.AssignedBy, &assignedAtStr, &completedAtStr, &o.EstimatedMinutes,
		&o.FulfillmentMinutes, &o.CancellationReason, &o.Rating, &o.Review, &createdStr,
	)
	if err != nil {
		return nil, err
	}

	o.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	if assignedAtStr != nil {
		t, err := time.Parse(time.RFC3339, *assignedAtStr)
		if err == nil {
			o.AssignedAt = &t
		}
	}
	if completedAtStr != nil {
		t, err := time.Parse(time.RFC3339, *completedAtStr)
		if err == nil {
			o.CompletedAt = &t
		}
	}
	return &o, nil
}

func scanOrderRow(row *sql.Row) (*models.Order, error) {
	var o models.Order
	var assignedAtStr, completedAtStr *string
	var createdStr string

	err := row.Scan(
		&o.ID, &o.UserID, &o.CustomerName, &o.CustomerPhone, &o.OrderType, &o.TableNumber, &o.DeliveryAddress,
		&o.Status, &o.PaymentMethod, &o.PaymentStatus, &o.TransactionID, &o.SubtotalPrice, &o.CouponCode,
		&o.DiscountAmount, &o.SubscriberDiscount, &o.SubscriberTierName, &o.TotalPrice, &o.AssignedStaffID,
		&o.AssignedStaffName, &o.AssignedBy, &assignedAtStr, &completedAtStr, &o.EstimatedMinutes,
		&o.FulfillmentMinutes, &o.CancellationReason, &o.Rating, &o.Review, &createdStr,
	)
	if err != nil {
		return nil, err
	}

	o.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	if assignedAtStr != nil {
		t, err := time.Parse(time.RFC3339, *assignedAtStr)
		if err == nil {
			o.AssignedAt = &t
		}
	}
	if completedAtStr != nil {
		t, err := time.Parse(time.RFC3339, *completedAtStr)
		if err == nil {
			o.CompletedAt = &t
		}
	}
	return &o, nil
}

// ------------------------------------------------------------------------------------------------
// AuditRepository Implementation
// ------------------------------------------------------------------------------------------------

func (r *SQLiteRepository) CreateAuditLog(ctx context.Context, log models.AuditLog) (*models.AuditLog, error) {
	ts := log.Timestamp.Format(time.RFC3339)
	if log.Timestamp.IsZero() {
		ts = time.Now().Format(time.RFC3339)
		log.Timestamp = time.Now()
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (timestamp, actor_id, actor_name, actor_role, action, details, ip_address)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, ts, log.ActorID, log.ActorName, log.ActorRole, log.Action, log.Details, log.IPAddress)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	log.ID = id
	return &log, nil
}

func (r *SQLiteRepository) GetAllAuditLogs(ctx context.Context) ([]models.AuditLog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, timestamp, actor_id, actor_name, actor_role, action, details, ip_address
		FROM audit_logs ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		var ts string
		if err := rows.Scan(&l.ID, &ts, &l.ActorID, &l.ActorName, &l.ActorRole, &l.Action, &l.Details, &l.IPAddress); err != nil {
			return nil, err
		}
		l.Timestamp, _ = time.Parse(time.RFC3339, ts)
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// ------------------------------------------------------------------------------------------------
// CouponRepository Implementation
// ------------------------------------------------------------------------------------------------

func (r *SQLiteRepository) CreateCoupon(ctx context.Context, coupon models.Coupon) (*models.Coupon, error) {
	coupon.Code = strings.ToUpper(strings.TrimSpace(coupon.Code))
	createdAt := coupon.CreatedAt.Format(time.RFC3339)
	if coupon.CreatedAt.IsZero() {
		createdAt = time.Now().Format(time.RFC3339)
		coupon.CreatedAt = time.Now()
	}
	expiryDate := coupon.ExpiryDate.Format(time.RFC3339)

	var usedAtStr *string
	if coupon.UsedAt != nil {
		s := coupon.UsedAt.Format(time.RFC3339)
		usedAtStr = &s
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO coupons (
			code, discount_type, discount_value, min_order_amount, expiry_date,
			target_user_id, target_user_name, is_used, used_at, used_by_order_id,
			created_by, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, coupon.Code, coupon.DiscountType, coupon.DiscountValue, coupon.MinOrderAmount, expiryDate,
		coupon.TargetUserID, coupon.TargetUserName, boolToInt(coupon.IsUsed), usedAtStr, coupon.UsedByOrderID,
		coupon.CreatedBy, createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			return nil, errors.New("coupon code already exists")
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	coupon.ID = id
	return &coupon, nil
}

func (r *SQLiteRepository) GetCouponByCode(ctx context.Context, code string) (*models.Coupon, error) {
	var c models.Coupon
	var expiryStr, createdStr string
	var usedAtStr *string
	var isUsedInt int

	err := r.db.QueryRowContext(ctx, `
		SELECT id, code, discount_type, discount_value, min_order_amount, expiry_date,
		       target_user_id, target_user_name, is_used, used_at, used_by_order_id,
		       created_by, created_at
		FROM coupons WHERE UPPER(code) = UPPER(?)
	`, strings.TrimSpace(code)).Scan(
		&c.ID, &c.Code, &c.DiscountType, &c.DiscountValue, &c.MinOrderAmount, &expiryStr,
		&c.TargetUserID, &c.TargetUserName, &isUsedInt, &usedAtStr, &c.UsedByOrderID,
		&c.CreatedBy, &createdStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("coupon not found")
	}
	if err != nil {
		return nil, err
	}

	c.IsUsed = (isUsedInt != 0)
	c.ExpiryDate, _ = time.Parse(time.RFC3339, expiryStr)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	if usedAtStr != nil {
		t, err := time.Parse(time.RFC3339, *usedAtStr)
		if err == nil {
			c.UsedAt = &t
		}
	}
	return &c, nil
}

func (r *SQLiteRepository) GetAllCoupons(ctx context.Context) ([]models.Coupon, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, discount_type, discount_value, min_order_amount, expiry_date,
		       target_user_id, target_user_name, is_used, used_at, used_by_order_id,
		       created_by, created_at
		FROM coupons ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coupons []models.Coupon
	for rows.Next() {
		var c models.Coupon
		var expiryStr, createdStr string
		var usedAtStr *string
		var isUsedInt int

		if err := rows.Scan(
			&c.ID, &c.Code, &c.DiscountType, &c.DiscountValue, &c.MinOrderAmount, &expiryStr,
			&c.TargetUserID, &c.TargetUserName, &isUsedInt, &usedAtStr, &c.UsedByOrderID,
			&c.CreatedBy, &createdStr,
		); err != nil {
			return nil, err
		}

		c.IsUsed = (isUsedInt != 0)
		c.ExpiryDate, _ = time.Parse(time.RFC3339, expiryStr)
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		if usedAtStr != nil {
			t, err := time.Parse(time.RFC3339, *usedAtStr)
			if err == nil {
				c.UsedAt = &t
			}
		}
		coupons = append(coupons, c)
	}
	return coupons, rows.Err()
}

func (r *SQLiteRepository) MarkCouponUsed(ctx context.Context, code string, orderID int64) error {
	now := time.Now().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx, `
		UPDATE coupons
		SET is_used = 1, used_at = ?, used_by_order_id = ?
		WHERE UPPER(code) = UPPER(?)
	`, now, orderID, strings.TrimSpace(code))
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("coupon not found")
	}
	return nil
}

func (r *SQLiteRepository) DeleteCoupon(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM coupons WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("coupon not found")
	}
	return nil
}

// ------------------------------------------------------------------------------------------------
// InventoryRepository Implementation
// ------------------------------------------------------------------------------------------------

func (r *SQLiteRepository) GetAllInventoryItems(ctx context.Context) ([]models.InventoryItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, category, item_name, unit, stock_quantity, reorder_level,
		       unit_cost, total_value, supplier, serial_number, status, updated_at
		FROM inventory_items ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.InventoryItem
	for rows.Next() {
		var itm models.InventoryItem
		var updatedStr string
		if err := rows.Scan(
			&itm.ID, &itm.Category, &itm.ItemName, &itm.Unit, &itm.StockQuantity, &itm.ReorderLevel,
			&itm.UnitCost, &itm.TotalValue, &itm.Supplier, &itm.SerialNumber, &itm.Status, &updatedStr,
		); err != nil {
			return nil, err
		}
		itm.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		items = append(items, itm)
	}
	return items, rows.Err()
}

func (r *SQLiteRepository) GetInventoryItemByID(ctx context.Context, id int64) (*models.InventoryItem, error) {
	var itm models.InventoryItem
	var updatedStr string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, category, item_name, unit, stock_quantity, reorder_level,
		       unit_cost, total_value, supplier, serial_number, status, updated_at
		FROM inventory_items WHERE id = ?
	`, id).Scan(
		&itm.ID, &itm.Category, &itm.ItemName, &itm.Unit, &itm.StockQuantity, &itm.ReorderLevel,
		&itm.UnitCost, &itm.TotalValue, &itm.Supplier, &itm.SerialNumber, &itm.Status, &updatedStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("inventory item not found")
	}
	if err != nil {
		return nil, err
	}
	itm.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &itm, nil
}

func (r *SQLiteRepository) SaveInventoryItem(ctx context.Context, item models.InventoryItem) (*models.InventoryItem, error) {
	item.TotalValue = item.StockQuantity * item.UnitCost
	if item.Status == "" {
		if item.Category == "Equipment" || item.Category == "Furniture" {
			item.Status = "Active Asset"
		} else if item.StockQuantity <= 0 {
			item.Status = "Out of Stock"
		} else if item.StockQuantity <= item.ReorderLevel {
			item.Status = "Low Stock"
		} else {
			item.Status = "In Stock"
		}
	}
	item.UpdatedAt = time.Now()
	updatedStr := item.UpdatedAt.Format(time.RFC3339)

	if item.ID > 0 {
		res, err := r.db.ExecContext(ctx, `
			UPDATE inventory_items
			SET category = ?, item_name = ?, unit = ?, stock_quantity = ?, reorder_level = ?,
			    unit_cost = ?, total_value = ?, supplier = ?, serial_number = ?, status = ?, updated_at = ?
			WHERE id = ?
		`, item.Category, item.ItemName, item.Unit, item.StockQuantity, item.ReorderLevel,
			item.UnitCost, item.TotalValue, item.Supplier, item.SerialNumber, item.Status, updatedStr, item.ID)
		if err != nil {
			return nil, err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rows == 0 {
			return nil, errors.New("inventory item not found")
		}
		return &item, nil
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO inventory_items (
			category, item_name, unit, stock_quantity, reorder_level,
			unit_cost, total_value, supplier, serial_number, status, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.Category, item.ItemName, item.Unit, item.StockQuantity, item.ReorderLevel,
		item.UnitCost, item.TotalValue, item.Supplier, item.SerialNumber, item.Status, updatedStr)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	item.ID = id
	return &item, nil
}

func (r *SQLiteRepository) UpdateStockQuantity(ctx context.Context, id int64, delta float64) error {
	itm, err := r.GetInventoryItemByID(ctx, id)
	if err != nil {
		return err
	}

	itm.StockQuantity += delta
	if itm.StockQuantity < 0 {
		itm.StockQuantity = 0
	}
	itm.TotalValue = itm.StockQuantity * itm.UnitCost
	if itm.Category != "Equipment" && itm.Category != "Furniture" {
		if itm.StockQuantity == 0 {
			itm.Status = "Out of Stock"
		} else if itm.StockQuantity <= itm.ReorderLevel {
			itm.Status = "Low Stock"
		} else {
			itm.Status = "In Stock"
		}
	}
	itm.UpdatedAt = time.Now()

	_, err = r.SaveInventoryItem(ctx, *itm)
	return err
}

func (r *SQLiteRepository) DeleteInventoryItem(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM inventory_items WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("inventory item not found")
	}
	return nil
}

func (r *SQLiteRepository) GetAllExpenses(ctx context.Context) ([]models.ExpenseEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, category, title, supplier_vendor, invoice_no, quantity,
		       unit_price, total_amount, payment_method, expense_date, notes, created_at
		FROM expenses ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.ExpenseEntry
	for rows.Next() {
		var exp models.ExpenseEntry
		var expDateStr, createdStr string
		if err := rows.Scan(
			&exp.ID, &exp.Category, &exp.Title, &exp.SupplierVendor, &exp.InvoiceNo, &exp.Quantity,
			&exp.UnitPrice, &exp.TotalAmount, &exp.PaymentMethod, &expDateStr, &exp.Notes, &createdStr,
		); err != nil {
			return nil, err
		}
		exp.ExpenseDate, _ = time.Parse(time.RFC3339, expDateStr)
		exp.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		expenses = append(expenses, exp)
	}
	return expenses, rows.Err()
}

func (r *SQLiteRepository) SaveExpense(ctx context.Context, expense models.ExpenseEntry) (*models.ExpenseEntry, error) {
	if expense.CreatedAt.IsZero() {
		expense.CreatedAt = time.Now()
	}
	if expense.ExpenseDate.IsZero() {
		expense.ExpenseDate = time.Now()
	}
	expDateStr := expense.ExpenseDate.Format(time.RFC3339)
	createdStr := expense.CreatedAt.Format(time.RFC3339)

	if expense.ID > 0 {
		res, err := r.db.ExecContext(ctx, `
			UPDATE expenses
			SET category = ?, title = ?, supplier_vendor = ?, invoice_no = ?, quantity = ?,
			    unit_price = ?, total_amount = ?, payment_method = ?, expense_date = ?, notes = ?
			WHERE id = ?
		`, expense.Category, expense.Title, expense.SupplierVendor, expense.InvoiceNo, expense.Quantity,
			expense.UnitPrice, expense.TotalAmount, expense.PaymentMethod, expDateStr, expense.Notes, expense.ID)
		if err != nil {
			return nil, err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rows == 0 {
			return nil, errors.New("expense not found")
		}
		return &expense, nil
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO expenses (
			category, title, supplier_vendor, invoice_no, quantity,
			unit_price, total_amount, payment_method, expense_date, notes, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, expense.Category, expense.Title, expense.SupplierVendor, expense.InvoiceNo, expense.Quantity,
		expense.UnitPrice, expense.TotalAmount, expense.PaymentMethod, expDateStr, expense.Notes, createdStr)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	expense.ID = id
	return &expense, nil
}

func (r *SQLiteRepository) DeleteExpense(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM expenses WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("expense not found")
	}
	return nil
}

// ------------------------------------------------------------------------------------------------
// SecurityRepository Implementation
// ------------------------------------------------------------------------------------------------

func (r *SQLiteRepository) GetAllAPIKeys(ctx context.Context) ([]models.APIKey, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, key_hash, key_prefix, role, is_active, last_used_at, created_at, expires_at
		FROM api_keys ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var k models.APIKey
		var isActInt int
		var createdStr string
		var lastUsedStr, expiresStr *string

		if err := rows.Scan(&k.ID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Role, &isActInt, &lastUsedStr, &createdStr, &expiresStr); err != nil {
			return nil, err
		}
		k.IsActive = (isActInt != 0)
		k.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		if lastUsedStr != nil {
			t, err := time.Parse(time.RFC3339, *lastUsedStr)
			if err == nil {
				k.LastUsedAt = &t
			}
		}
		if expiresStr != nil {
			t, err := time.Parse(time.RFC3339, *expiresStr)
			if err == nil {
				k.ExpiresAt = &t
			}
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *SQLiteRepository) GetAPIKeyByID(ctx context.Context, id int64) (*models.APIKey, error) {
	var k models.APIKey
	var isActInt int
	var createdStr string
	var lastUsedStr, expiresStr *string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, key_hash, key_prefix, role, is_active, last_used_at, created_at, expires_at
		FROM api_keys WHERE id = ?
	`, id).Scan(&k.ID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Role, &isActInt, &lastUsedStr, &createdStr, &expiresStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("api key not found")
	}
	if err != nil {
		return nil, err
	}
	k.IsActive = (isActInt != 0)
	k.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	if lastUsedStr != nil {
		t, err := time.Parse(time.RFC3339, *lastUsedStr)
		if err == nil {
			k.LastUsedAt = &t
		}
	}
	if expiresStr != nil {
		t, err := time.Parse(time.RFC3339, *expiresStr)
		if err == nil {
			k.ExpiresAt = &t
		}
	}
	return &k, nil
}

func (r *SQLiteRepository) GetAPIKeyByHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	var k models.APIKey
	var isActInt int
	var createdStr string
	var lastUsedStr, expiresStr *string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, key_hash, key_prefix, role, is_active, last_used_at, created_at, expires_at
		FROM api_keys WHERE key_hash = ?
	`, keyHash).Scan(&k.ID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Role, &isActInt, &lastUsedStr, &createdStr, &expiresStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("api key not found")
	}
	if err != nil {
		return nil, err
	}
	k.IsActive = (isActInt != 0)
	k.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	if lastUsedStr != nil {
		t, err := time.Parse(time.RFC3339, *lastUsedStr)
		if err == nil {
			k.LastUsedAt = &t
		}
	}
	if expiresStr != nil {
		t, err := time.Parse(time.RFC3339, *expiresStr)
		if err == nil {
			k.ExpiresAt = &t
		}
	}
	return &k, nil
}

func (r *SQLiteRepository) SaveAPIKey(ctx context.Context, apiKey models.APIKey) (*models.APIKey, error) {
	if apiKey.CreatedAt.IsZero() {
		apiKey.CreatedAt = time.Now()
	}
	createdStr := apiKey.CreatedAt.Format(time.RFC3339)
	var lastUsedStr, expiresStr *string
	if apiKey.LastUsedAt != nil {
		s := apiKey.LastUsedAt.Format(time.RFC3339)
		lastUsedStr = &s
	}
	if apiKey.ExpiresAt != nil {
		s := apiKey.ExpiresAt.Format(time.RFC3339)
		expiresStr = &s
	}

	if apiKey.ID > 0 {
		res, err := r.db.ExecContext(ctx, `
			UPDATE api_keys
			SET name = ?, key_hash = ?, key_prefix = ?, role = ?, is_active = ?, last_used_at = ?, expires_at = ?
			WHERE id = ?
		`, apiKey.Name, apiKey.KeyHash, apiKey.KeyPrefix, apiKey.Role, boolToInt(apiKey.IsActive), lastUsedStr, expiresStr, apiKey.ID)
		if err != nil {
			return nil, err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rows == 0 {
			return nil, errors.New("api key not found")
		}
		return &apiKey, nil
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO api_keys (name, key_hash, key_prefix, role, is_active, last_used_at, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, apiKey.Name, apiKey.KeyHash, apiKey.KeyPrefix, apiKey.Role, boolToInt(apiKey.IsActive), lastUsedStr, createdStr, expiresStr)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	apiKey.ID = id
	return &apiKey, nil
}

func (r *SQLiteRepository) UpdateAPIKeyLastUsed(ctx context.Context, id int64) error {
	now := time.Now().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx, "UPDATE api_keys SET last_used_at = ? WHERE id = ?", now, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("api key not found")
	}
	return nil
}

func (r *SQLiteRepository) RevokeAPIKey(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "UPDATE api_keys SET is_active = 0 WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("api key not found")
	}
	return nil
}

// ------------------------------------------------------------------------------------------------
// MembershipRepository Implementation
// ------------------------------------------------------------------------------------------------

func (r *SQLiteRepository) GetAllSubscriptions(ctx context.Context) ([]models.UserSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, user_name, user_email, tier_id, tier_name, status,
		       price_paid, discount_percent, daily_free_cup_limit, cups_claimed_today,
		       last_claim_date, start_date, end_date, auto_renew, payment_method, transaction_id
		FROM subscriptions ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.UserSubscription
	for rows.Next() {
		var s models.UserSubscription
		var autoRenewInt int
		var startStr, endStr string
		var lastClaimStr *string

		if err := rows.Scan(
			&s.ID, &s.UserID, &s.UserName, &s.UserEmail, &s.TierID, &s.TierName, &s.Status,
			&s.PricePaid, &s.DiscountPercent, &s.DailyFreeCupLimit, &s.CupsClaimedToday,
			&lastClaimStr, &startStr, &endStr, &autoRenewInt, &s.PaymentMethod, &s.TransactionID,
		); err != nil {
			return nil, err
		}

		s.AutoRenew = (autoRenewInt != 0)
		s.StartDate, _ = time.Parse(time.RFC3339, startStr)
		s.EndDate, _ = time.Parse(time.RFC3339, endStr)
		if lastClaimStr != nil {
			t, err := time.Parse(time.RFC3339, *lastClaimStr)
			if err == nil {
				s.LastClaimDate = &t
			}
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (r *SQLiteRepository) GetSubscriptionByUserID(ctx context.Context, userID int64) (*models.UserSubscription, error) {
	var s models.UserSubscription
	var autoRenewInt int
	var startStr, endStr string
	var lastClaimStr *string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, user_name, user_email, tier_id, tier_name, status,
		       price_paid, discount_percent, daily_free_cup_limit, cups_claimed_today,
		       last_claim_date, start_date, end_date, auto_renew, payment_method, transaction_id
		FROM subscriptions WHERE user_id = ? AND status = 'Active'
		ORDER BY id DESC LIMIT 1
	`, userID).Scan(
		&s.ID, &s.UserID, &s.UserName, &s.UserEmail, &s.TierID, &s.TierName, &s.Status,
		&s.PricePaid, &s.DiscountPercent, &s.DailyFreeCupLimit, &s.CupsClaimedToday,
		&lastClaimStr, &startStr, &endStr, &autoRenewInt, &s.PaymentMethod, &s.TransactionID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("no active subscription found for user")
	}
	if err != nil {
		return nil, err
	}

	s.AutoRenew = (autoRenewInt != 0)
	s.StartDate, _ = time.Parse(time.RFC3339, startStr)
	s.EndDate, _ = time.Parse(time.RFC3339, endStr)
	if lastClaimStr != nil {
		t, err := time.Parse(time.RFC3339, *lastClaimStr)
		if err == nil {
			s.LastClaimDate = &t
		}
	}
	return &s, nil
}

func (r *SQLiteRepository) GetSubscriptionByID(ctx context.Context, id int64) (*models.UserSubscription, error) {
	var s models.UserSubscription
	var autoRenewInt int
	var startStr, endStr string
	var lastClaimStr *string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, user_name, user_email, tier_id, tier_name, status,
		       price_paid, discount_percent, daily_free_cup_limit, cups_claimed_today,
		       last_claim_date, start_date, end_date, auto_renew, payment_method, transaction_id
		FROM subscriptions WHERE id = ?
	`, id).Scan(
		&s.ID, &s.UserID, &s.UserName, &s.UserEmail, &s.TierID, &s.TierName, &s.Status,
		&s.PricePaid, &s.DiscountPercent, &s.DailyFreeCupLimit, &s.CupsClaimedToday,
		&lastClaimStr, &startStr, &endStr, &autoRenewInt, &s.PaymentMethod, &s.TransactionID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("subscription not found")
	}
	if err != nil {
		return nil, err
	}

	s.AutoRenew = (autoRenewInt != 0)
	s.StartDate, _ = time.Parse(time.RFC3339, startStr)
	s.EndDate, _ = time.Parse(time.RFC3339, endStr)
	if lastClaimStr != nil {
		t, err := time.Parse(time.RFC3339, *lastClaimStr)
		if err == nil {
			s.LastClaimDate = &t
		}
	}
	return &s, nil
}

func (r *SQLiteRepository) SaveSubscription(ctx context.Context, sub models.UserSubscription) (*models.UserSubscription, error) {
	if sub.StartDate.IsZero() {
		sub.StartDate = time.Now()
	}
	if sub.EndDate.IsZero() {
		sub.EndDate = sub.StartDate.AddDate(0, 1, 0)
	}
	if sub.Status == "" {
		sub.Status = "Active"
	}

	startStr := sub.StartDate.Format(time.RFC3339)
	endStr := sub.EndDate.Format(time.RFC3339)
	var lastClaimStr *string
	if sub.LastClaimDate != nil {
		s := sub.LastClaimDate.Format(time.RFC3339)
		lastClaimStr = &s
	}

	if sub.ID > 0 {
		res, err := r.db.ExecContext(ctx, `
			UPDATE subscriptions
			SET user_id = ?, user_name = ?, user_email = ?, tier_id = ?, tier_name = ?, status = ?,
			    price_paid = ?, discount_percent = ?, daily_free_cup_limit = ?, cups_claimed_today = ?,
			    last_claim_date = ?, start_date = ?, end_date = ?, auto_renew = ?, payment_method = ?, transaction_id = ?
			WHERE id = ?
		`, sub.UserID, sub.UserName, sub.UserEmail, sub.TierID, sub.TierName, sub.Status,
			sub.PricePaid, sub.DiscountPercent, sub.DailyFreeCupLimit, sub.CupsClaimedToday,
			lastClaimStr, startStr, endStr, boolToInt(sub.AutoRenew), sub.PaymentMethod, sub.TransactionID, sub.ID)
		if err != nil {
			return nil, err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rows == 0 {
			return nil, errors.New("subscription not found")
		}
		return &sub, nil
	}

	// Deactivate any previous active subscriptions for this user
	_, _ = r.db.ExecContext(ctx, "UPDATE subscriptions SET status = 'Cancelled' WHERE user_id = ? AND status = 'Active'", sub.UserID)

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO subscriptions (
			user_id, user_name, user_email, tier_id, tier_name, status,
			price_paid, discount_percent, daily_free_cup_limit, cups_claimed_today,
			last_claim_date, start_date, end_date, auto_renew, payment_method, transaction_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sub.UserID, sub.UserName, sub.UserEmail, sub.TierID, sub.TierName, sub.Status,
		sub.PricePaid, sub.DiscountPercent, sub.DailyFreeCupLimit, sub.CupsClaimedToday,
		lastClaimStr, startStr, endStr, boolToInt(sub.AutoRenew), sub.PaymentMethod, sub.TransactionID)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	sub.ID = id
	return &sub, nil
}

func (r *SQLiteRepository) ClaimDailyCup(ctx context.Context, subscriptionID int64) error {
	sub, err := r.GetSubscriptionByID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	now := time.Now()
	// Reset claimed cups if new calendar day
	if sub.LastClaimDate == nil || sub.LastClaimDate.Format("2006-01-02") != now.Format("2006-01-02") {
		sub.CupsClaimedToday = 0
	}

	if sub.CupsClaimedToday >= sub.DailyFreeCupLimit {
		return errors.New("daily free cup allowance limit reached")
	}

	sub.CupsClaimedToday++
	sub.LastClaimDate = &now

	_, err = r.SaveSubscription(ctx, *sub)
	return err
}

func (r *SQLiteRepository) CancelSubscription(ctx context.Context, subscriptionID int64) error {
	res, err := r.db.ExecContext(ctx, "UPDATE subscriptions SET status = 'Cancelled' WHERE id = ?", subscriptionID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("subscription not found")
	}
	return nil
}

// ------------------------------------------------------------------------------------------------
// CafeSettingsRepository Implementation
// ------------------------------------------------------------------------------------------------

func (r *SQLiteRepository) GetCafeSettings(ctx context.Context) (*models.CafeSettings, error) {
	var s models.CafeSettings
	var annEnabledInt int

	err := r.db.QueryRowContext(ctx, `
		SELECT store_name, store_address, brewing_hours, store_phone, currency_symbol,
		       announcement_enabled, announcement_text, announcement_phone
		FROM cafe_settings WHERE id = 1
	`).Scan(&s.StoreName, &s.StoreAddress, &s.BrewingHours, &s.StorePhone, &s.CurrencySymbol,
		&annEnabledInt, &s.AnnouncementText, &s.AnnouncementPhone)

	if errors.Is(err, sql.ErrNoRows) {
		s = models.CafeSettings{
			StoreName:           "TEACHAR Flagship Cafe Sanctuary",
			StoreAddress:        "42 Chai Galleria, MG Road, Tech Hub District, Bangalore, 560001",
			BrewingHours:        "6:00 AM – 11:30 PM",
			StorePhone:          "+91 98765 43210",
			CurrencySymbol:      "₹",
			AnnouncementEnabled: true,
			AnnouncementText:    "Get 20% OFF your first order! Use code FRESHTEA",
			AnnouncementPhone:   "+91 98765 43210",
		}
		_ = r.UpdateCafeSettings(ctx, s)
		return &s, nil
	}
	if err != nil {
		return nil, err
	}

	s.AnnouncementEnabled = (annEnabledInt != 0)
	return &s, nil
}

func (r *SQLiteRepository) UpdateCafeSettings(ctx context.Context, settings models.CafeSettings) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cafe_settings (
			id, store_name, store_address, brewing_hours, store_phone, currency_symbol,
			announcement_enabled, announcement_text, announcement_phone
		) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			store_name = excluded.store_name,
			store_address = excluded.store_address,
			brewing_hours = excluded.brewing_hours,
			store_phone = excluded.store_phone,
			currency_symbol = excluded.currency_symbol,
			announcement_enabled = excluded.announcement_enabled,
			announcement_text = excluded.announcement_text,
			announcement_phone = excluded.announcement_phone
	`, settings.StoreName, settings.StoreAddress, settings.BrewingHours, settings.StorePhone, settings.CurrencySymbol,
		boolToInt(settings.AnnouncementEnabled), settings.AnnouncementText, settings.AnnouncementPhone)
	return err
}

// ------------------------------------------------------------------------------------------------
// Helpers & Password Security Utilities
// ------------------------------------------------------------------------------------------------

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// GenerateSalt creates a cryptographically secure random 16-byte hex salt string.
func GenerateSalt() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// HashPassword hashes a password string with a salt using SHA-256.
func HashPassword(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(password + salt))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateToken creates a cryptographically secure random session token string.
func GenerateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("token_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

