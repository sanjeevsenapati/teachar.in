# TEACHAR.in

A modern, high-performance, and responsive web application for **TEACHAR**, built using **Go** and an embedded CGO-free **SQLite** database (`modernc.org/sqlite`).

---

## 📸 Visual Showcase & Menu Gallery

### 🏠 Homepage Showcase & Artisanal Hero Banner
![TEACHAR Home Screen](static/images/home-screenshot.png)

---

### 🍵 Signature Teas & Authentic Indian Brews

| Kulhad Masala Chai | Red Tea & Popped Rice Combo | Fresh Ginger Tea |
|:---:|:---:|:---:|
| ![Kulhad Masala Chai](static/images/hero-banner.jpg) | ![Red Tea Combo](static/images/red-tea-combo.jpg) | ![Ginger Tea](static/images/ginger-tea.jpg) |
| *Signature Spiced Kulhad Chai* | *Red Tea & Popped Rice Ritual* | *Fresh Crushed Ginger Tea* |

---

### 🥐 Artisanal Snacks & Beverages

| Golden Masala Samosa | Iced Cold Coffee | Grilled Veg Sandwich |
|:---:|:---:|:---:|
| ![Golden Samosa](static/images/samosa.jpg) | ![Cold Coffee](static/images/cold-coffee.jpg) | ![Veg Sandwich](static/images/veg-sandwich.jpg) |
| *Crispy Handcrafted Samosa* | *Creamy Espresso Cold Brew* | *Fresh Grilled Sandwich* |

---

## 🌟 Overview

**TEACHAR.in** is a modern artisanal beverage and gastronomy platform logically separated into two distinct, high-performance web applications powered by a single embedded **SQLite** database (`data/teachar.db` with WAL mode):

1. **🍵 Public Customer Web App** (`https://teachar.in:8080` / `https://localhost:8080`):
   - Fast, elegant customer storefront: Artisan Menu (`/menu`), About (`/about`), Membership (`/membership`), Customer Auth (`/login`, `/register`), 2-Column Checkout Drawer, Multi-Channel Payment Gateway, Live Order Tracking (`/orders`), and Customer Reviews.
   - Admin routes are completely unexposed and blocked (returning HTTP 404).

2. **🔐 Private Staff & Admin Portal** (`https://admin.teachar.in:8081` / `https://localhost:8081`):
   - Dedicated workplace for Counter Staff, Cafe Admins, and SuperAdmins.
   - Live Order Queue & Staff claim tagging, Menu Management, Staff Speed & SLA Compliance Analytics, Store Inventory & Asset Register, Operating Expenditure Log, Financial Year Tax Audit Statements, User/Staff Management, Promo Coupons, System Audit Trails, API Keys, and Store Settings.

---

## 🚀 Comprehensive Feature List

### 🍵 Customer Features
- **Artisanal Menu (`/menu`)**: Browse categorised items (*Tea*, *Coffee*, *Snacks*, *Cold Beverages*) with live instant search and category filtering tabs.
- **Enhanced 2-Column Checkout Drawer**:
  - **Spacious Order Summary**: Clear view of items, quantities, coupons/offers, and price breakdown.
  - **Un-Squeezed Fulfillment Selectors**: Smooth horizontal pill selectors for *Dine-in*, *Takeaway*, and *Delivery* with dynamic table number, mobile number, and address fields.
  - **Multi-Channel Payment Gateway**:
    - **UPI / QR Instant Pay**: Dynamic QR code preview (`teachar@upi`), auto-generated VPA string, and instant authorization.
    - **Credit / Debit Card**: Formatted card inputs with CVV validation and interactive **3D Secure OTP Modal** verification.
    - **NetBanking**: Major Indian banks (HDFC, ICICI, SBI, Axis, Kotak).
    - **Cash on Delivery (COD)**: Pay at table or on delivery (*Pending COD*).
    - **Tax Calculation**: Automated 5% GST tax breakdown.
- **Customer Authentication**: Secure sign-up (`/register`), sign-in (`/login`), and sign-out (`/logout`).
- **Order & Receipt Tracking (`/orders`)**: Monitor order fulfillment status (*Pending*, *Preparing*, *Ready*, *Completed*, *Cancelled*), view Payment Method pills (*UPI*, *Card*, *COD*), Payment Status badges (*Paid*, *Pending COD*), Staff Claim tags, Cancellation reasons, and unique Transaction IDs (`TXN...`).
- **Interactive Rating & Review System**: Once an order reaches *Completed* status, customers can submit a **1 to 5 Star Rating** and written review directly on their order receipt.
- **User Profile (`/account`)**: Manage personal account information and view purchase statistics.

---

### 🔒 Security, TLS/SSL & API Key System

#### 🔑 API Key Authentication (`/admin/api-keys`)
- **Key Generation & SHA-256 Hashing**: Issue cryptographically secure secret tokens (`tch_live_<32 hex bytes>`). Secret keys are displayed **ONCE** upon generation, with SHA-256 hashes stored securely in the SQLite database (`api_keys` table).
- **Standard Auth Headers**: Authenticate REST requests via `X-API-Key: tch_live_...` or `Authorization: Bearer tch_live_...`.
- **Superadmin API Portal**: Issue, inspect, and revoke API keys for external POS terminals, partner platforms, or mobile clients.

#### 🛡️ TLS / HTTPS / SSL Engine
- **Dual Server Support**: Configurable via `ENABLE_TLS`, `SSL_CERT_FILE`, `SSL_KEY_FILE`, and `SSL_PORT` environment variables.
- **Automated Certificate Generator**: Built-in ECDSA/RSA P-256 TLS certificate generator (`services.GenerateSelfSignedCert()`) using `crypto/x509` and `crypto/rsa` for local HTTPS testing (`https://localhost:8443`).

#### ⚡ Rate Limiting & Security Hardening
- **Sliding Window Rate Limiter (`middleware/rate_limiter.go`)**: Thread-safe IP & API-Key rate limiting preventing DDoS attacks, brute-force logins, and API scraping (returns `HTTP 429 Too Many Requests` with `Retry-After: 60`).
- **Hardened Security Headers**:
  - `Strict-Transport-Security: max-age=63072000; includeSubDomains; preload` (HSTS)
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: deny`
  - `X-XSS-Protection: 1; mode=block`
  - `Referrer-Policy: origin-when-cross-origin`
  - `Permissions-Policy: camera=(), microphone=(), geolocation=()`

---

### 🛡️ Admin, Staff & Superadmin Features

#### 👥 Staff Portal (`/admin/orders`)
- **Real-Time Live Search**: Search orders instantly by Table Number, Customer Registered Mobile Number, Customer Name, or Order/TXN ID.
- **Staff-Defined Estimated Prep Duration**: Staff/Admin inputs estimated prep duration (`10m`, `15m`, `20m`, `25m`, `30m`) when picking up or assigning an order. Customer receipts display clean status updates without internal timers.
- **Duplicate-Order Protection (Staff Claim Tagging)**: Automatically tagged with Staff Name & ID (`AssignedStaffID`, `AssignedStaffName`), preventing duplicate processing.
- **Superadmin Order Assignment Privilege**: Superadmins and Admins can assign unclaimed orders to specific staff members with custom prep durations.
- **Mandatory Order Cancellation Reasons**: Select predefined cancellation reasons from a dropdown list or enter custom reasons.
- **Printable & Downloadable GST Tax Invoices**: Thermal printable receipt (`window.print()`) and PDF download support (`downloadBillAsPDF()`).

#### 👑 Superadmin & Executive Management Portal (`/admin`)
- **Role-Based Access Control (RBAC)**: Support for `superadmin`, `admin`, `staff`, and `client` roles.
- **Executive Dashboard (`/admin`)**: Business metrics including Total Revenue (₹), Total Orders Count, Active Orders Count, and Total Menu Items.
- **Store Inventory & Capital Asset Register (`/admin/inventory`)**:
  - Track **Raw Materials**, **Consumables**, **Machinery & Equipment**, and **Furniture & Fixtures**.
  - Real-time stock levels, unit costs, reorder alert thresholds (`In Stock`, `Low Stock`, `Out of Stock`, `Active Asset`), and serial number tracking.
  - Side-by-side 4-column horizontal card grid matching TEACHAR.in warm cream & artisanal terracotta theme.
- **Operating Expenditure Log (`/admin/expenses`)**: Track rent, electricity/water utilities, equipment maintenance, transportation/freight, and raw material purchases.
- **Financial Year Tax Audit Engine & CSV Export (`/admin/inventory/export`)**:
  - Automated financial audit metrics: Gross Sales Revenue, COGS, OpEx, Capital Asset Valuation, Net EBITDA Profit/Loss, 5% Output GST, and 25% Income Tax estimates.
  - **One-Click CA Audit CSV Export**: Download itemized audit spreadsheets for tax filings (`teachar_store_inventory_tax_audit_fy.csv`).
- **Staff Speed, SLA & Satisfaction Analytics (`/admin/staff-performance`)**: Track staff prep speed against custom SLAs, on-time fulfillment rates, and customer 5-star ratings.
- **API Keys & Security Portal (`/admin/api-keys`)**: Issue and revoke API authentication keys for developer integrations.
- **Offers & Coupon Management (`/admin/coupons`)**: Create, validate, and manage discount promo codes.
- **Staff & Admin User Management (`/admin/users`)**: Create and manage staff/admin user accounts.
- **System Audit Logs Trail (`/admin/audit-logs`)**: Immutable audit trail logging user registrations, logins, status changes, inventory additions, expense records, and API key generation.
- **Menu Item Management (`/admin/menu`)**: Add menu items with image uploads, edit details, toggle availability (*Available* / *Sold Out*), or delete items.

---

## 🔑 Demo Accounts

For testing all roles and user flows:

| Role | Email | Password | Access Level |
|---|---|---|---|
| **Superadmin** | `superadmin@teachar.in` | `SuperAdmin@123` | Full System Access, API Keys, Store Inventory, Tax Audit, User Management, Audit Logs |
| **Admin** | `admin@teachar.in` | `Admin@123` | Executive Dashboard, Order Management, Menu Management, Store Inventory & Expenses |
| **Staff** | `staff@teachar.in` | `Staff@123` | Order Search, Order Claim Tagging, Custom Prep SLA Input, Status Updates, Bill Generation |
| **Client** | `client@teachar.in` | `Client@123` | Menu Browsing, Checkout, Order Tracking, Rating & Review |

---

## ⚡ High-Performance Architecture

The application uses an **Embedded SQLite Architecture** (`SQLiteRepository`) with Write-Ahead Logging (WAL mode) engineered for high concurrency, fast query performance, and ACID transactional integrity using Go's `database/sql` interface and `modernc.org/sqlite`:

```text
HTTP Request
     ↓
Middleware Layer (Rate Limiter, Security Headers, HSTS, Session Authentication, API Key Auth, Role Authorization)
     ↓
Handlers Layer (Web Pages, Auth, Client Portal, Staff Order Search, Admin Portal, Store Inventory, API Keys, Payment APIs)
     ↓
Services Layer (MenuService, AuthService, OrderService, AuditService, ReportService, InventoryService, SecurityService, CouponService)
     ↓
SQLite Repository Layer (SQLiteRepository with Prepared Statements & Transaction Management)
     ↓
Embedded SQLite Database (data/teachar.db with WAL mode & foreign key constraints)
```

### High-Concurrency & Reliability Features:
- **Embedded SQLite Engine**: Persistent storage via `data/teachar.db` with Write-Ahead Logging (`journal_mode=WAL`) for concurrent reads and writes.
- **ACID Transaction Support**: Atomic transaction boundaries for multi-table updates (e.g., order creation alongside coupon validation and inventory tracking).
- **Indexed Relational Schema**: SQL tables (`users`, `sessions`, `orders`, `menu_items`, `inventory`, `expenses`, `api_keys`, `audit_logs`, `coupons`) with B-tree indexes for fast sub-millisecond lookups.
- **Auto-Migrating Schema**: Automatic DDL table creation and schema migration on server startup.
- **Hardened Security & Prepared Statements**: Complete protection against SQL injection vulnerabilities using parameterized queries.

---

## 📁 File & Project Directory Structure

```text
teachar.in/
├── main.go                 # Application entry point, TLS HTTPS server & service initialization
├── README.MD               # Project documentation
│
├── config/                 # Environment & TLS configuration loading
│   ├── config.go
│   └── config_test.go
│
├── models/                 # Data structures (MenuItem, User, Session, Order, InventoryItem, ExpenseEntry, APIKey, AuditLog)
│   ├── menu.go
│   ├── inventory.go
│   ├── security.go
│   └── store.go
│
├── repository/             # SQLite Persistent Storage Layer (ACID & WAL mode)
│   ├── sqlite_repository.go
│   ├── sqlite_repository_test.go
│   ├── menu_repository.go
│   ├── inventory_repository.go
│   ├── membership_repository.go
│   └── security_repository.go
│
├── services/               # Business logic layer
│   ├── menu_service.go
│   ├── auth_service.go
│   ├── order_service.go
│   ├── inventory_service.go
│   ├── security_service.go
│   ├── security_service_test.go
│   ├── audit_service.go
│   ├── report_service.go
│   ├── coupon_service.go
│   └── menu_service_test.go
│
├── middleware/             # HTTP middleware (Rate Limiter, Security Headers, HSTS, Auth, RequireAPIKey, RequireAdmin, RequireRole)
│   ├── middleware.go
│   ├── rate_limiter.go
│   ├── rate_limiter_test.go
│   └── middleware_test.go
│
├── handlers/               # HTTP handlers for web pages, client, auth, staff, admin, API keys, store inventory & tax audit
│   ├── handlers.go
│   ├── pages.go
│   ├── api.go
│   ├── auth.go
│   ├── client.go
│   ├── admin.go
│   ├── inventory.go
│   ├── security.go
│   └── handlers_test.go
│
├── templates/              # HTML5 Semantic Templates
│   ├── layout.html         # Main layout & 2-column checkout drawer
│   ├── home.html           # Home page
│   ├── menu.html           # Full menu page
│   ├── about.html          # About Us page
│   ├── login.html          # Sign In page
│   ├── register.html       # Sign Up page
│   ├── client_orders.html  # Order history & Rating & Review form
│   ├── client_account.html # User profile page
│   ├── admin_dashboard.html# Executive dashboard
│   ├── admin_menu.html     # Menu management table
│   ├── admin_users.html    # Staff/Admin user management & Cancellation reasons
│   ├── admin_orders.html   # Live order search, Claim tagging, prep SLA & GST Bill modal
│   ├── admin_inventory.html# Store Inventory & Capital Asset Register
│   ├── admin_expenses.html # Operating Expenditure Log & Tax Audit Statement
│   ├── admin_staff_performance.html # Staff speed, SLA compliance & 5-star ratings board
│   ├── admin_api_keys.html # API Authentication Keys & Developer Access portal
│   ├── admin_reports.html  # Executive sales analytics & period filters
│   └── admin_coupons.html  # Promo codes & discount coupon management
│
├── static/                 # Static Assets
│   ├── style.css           # CSS design system, horizontal card grids & print receipt styles
│   ├── app.js              # Client JS cart, payment selector & OTP modal
│   └── images/             # Food & beverage assets
└── data/                   # Embedded Database & Security Assets
    ├── teachar.db          # Embedded SQLite Database (Tables, Indexes, WAL Mode)
    └── certs/              # SSL/TLS Certificates
```

---

## ⚙️ Requirements

- **Go 1.22** or newer.

---

## 🛠️ Running Locally

1. Clone the repository and navigate to the project directory:
   ```bash
   cd teachar.in
   ```

2. Run the application:
   ```bash
   go run .
   ```

3. Open your browser and navigate to:
   **[http://localhost:8080](http://localhost:8080)**

---

## 🧪 Running Tests & Concurrency Benchmarks

To run the entire automated unit test suite and high-concurrency race detector:

```bash
go test -v -race ./...
```

---

## 📦 Building for Production

To compile a single, self-contained binary:

```bash
go build -o teachar_server .
```

Run the compiled executable:

```bash
./teachar_server
```

---

## ⚙️ Configuration

The application is configured using environment variables with sensible defaults:

| Variable | Description | Default |
|---|---|---|
| `APP_NAME` | Name of the application | `teachar.in` |
| `APP_HOST` | Host address to bind to | `0.0.0.0` |
| `APP_PORT` | Port number to listen on | `8080` |
| `APP_ENV` | Application environment | `development` |
| `ENABLE_TLS` | Enable HTTPS TLS Server | `false` |
| `SSL_CERT_FILE` | Path to TLS certificate PEM file | `data/certs/cert.pem` |
| `SSL_KEY_FILE` | Path to TLS private key PEM file | `data/certs/key.pem` |
| `SSL_PORT` | HTTPS port number | `8443` |

---

## 📜 License

Distributed under the MIT License.
