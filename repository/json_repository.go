package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"teachar.in/models"
)

type dbSchema struct {
	NextUserID     int64             `json:"next_user_id"`
	NextMenuItemID int64             `json:"next_menu_item_id"`
	NextOrderID    int64             `json:"next_order_id"`
	Users          []models.User     `json:"users"`
	Sessions       []models.Session  `json:"sessions"`
	MenuItems      []models.MenuItem `json:"menu_items"`
	Orders         []models.Order    `json:"orders"`
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
	adminSalt := GenerateSalt()
	clientSalt := GenerateSalt()

	r.data = dbSchema{
		NextUserID:     3,
		NextMenuItemID: 13,
		NextOrderID:    1,
		Users: []models.User{
			{
				ID:           1,
				Name:         "TEACHAR Admin",
				Email:        "admin@teachar.in",
				PasswordHash: HashPassword("Admin@123", adminSalt),
				Salt:         adminSalt,
				Role:         "admin",
				CreatedAt:    time.Now(),
			},
			{
				ID:           2,
				Name:         "Sample Client",
				Email:        "client@teachar.in",
				PasswordHash: HashPassword("Client@123", clientSalt),
				Salt:         clientSalt,
				Role:         "client",
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
		},
		Orders: []models.Order{},
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
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, o := range r.data.Orders {
		if o.ID == id {
			r.data.Orders[i].Status = status
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("order not found")
	}
	return r.save()
}
