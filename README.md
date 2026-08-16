# TEACHAR.in

A modern, high-performance, and responsive web application for **TEACHAR**, built using **100% Pure Go Standard Library** (zero third-party dependencies).

---

## 📸 Home Screen

![TEACHAR Home Screen](static/images/home-screenshot.png)

---

## 🌟 Overview

**TEACHAR.in** is a digital platform for an artisanal tea, coffee, and gourmet snack house. It demonstrates the power, performance, and completeness of Go's standard library for web development—featuring domain-isolated persistent JSON storage, cryptographic authentication, role-based access control, an interactive customer shopping cart, multi-channel payment gateway, real-time order tracking, live staff order search, printable GST tax invoices, customer rating & review system, store inventory & capital asset register, operating expenditure log, financial year tax audit reports, and a comprehensive superadmin executive dashboard.

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

### 🛡️ Admin, Staff & Superadmin Features

#### 👥 Staff Portal (`/admin/orders`)
- **Real-Time Live Search**: Search orders instantly by Table Number, Customer Registered Mobile Number, Customer Name, or Order/TXN ID.
- **Staff-Defined Estimated Prep Duration**: Staff/Admin inputs estimated prep duration (`10m`, `15m`, `20m`, `25m`, `30m`) when picking up or assigning an order. Customer receipts display clean status updates without internal timers.
- **Duplicate-Order Protection (Staff Claim Tagging)**: When a staff member picks up or updates an order, it is automatically tagged with the Staff's Name and Staff ID (`AssignedStaffID`, `AssignedStaffName`), preventing duplicate processing.
- **Superadmin Order Assignment Privilege**: Superadmins and Admins can assign unclaimed orders to specific staff members with custom prep durations. Superadmins are restricted from directly fulfilling orders themselves.
- **Mandatory Order Cancellation Reasons**: When marking an order as *Cancelled*, staff must select a reason from a predefined dropdown list or enter a custom reason.
- **Printable & Downloadable GST Tax Invoices**:
  - Click **"Generate Bill"** on any order row to open a formatted **GST Tax Invoice Modal** with 5% GST breakdown, subtotal, and net payable amount.
  - **Thermal Print & PDF Download**: Printable thermal receipt (`window.print()`) and PDF download support (`downloadBillAsPDF()`).

#### 👑 Superadmin & Executive Management Portal (`/admin`)
- **Role-Based Access Control (RBAC)**: Support for `superadmin`, `admin`, `staff`, and `client` roles with strict middleware enforcement.
- **Executive Dashboard (`/admin`)**: Business metrics including Total Revenue (₹), Total Orders Count, Active Orders Count, and Total Menu Items.
- **Store Inventory & Capital Asset Register (`/admin/inventory`)**:
  - Track **Raw Materials** (Tea, Coffee, Sugar, Milk, Spices), **Consumables**, **Machinery & Equipment**, and **Furniture & Fixtures**.
  - Real-time stock levels, unit costs, reorder alert thresholds (`In Stock`, `Low Stock`, `Out of Stock`, `Active Asset`), and serial number tracking.
  - Formatted in a side-by-side 4-column horizontal card grid matching TEACHAR.in warm cream & artisanal terracotta theme.
- **Operating Expenditure Log (`/admin/expenses`)**:
  - Track cafe rent, electricity & water utilities, equipment maintenance, transportation/freight, and raw material purchases.
  - Log vendor/supplier names, invoice numbers, payment methods (Bank Transfer, UPI, Cash, Cheque, Card), and expense dates.
- **Financial Year Tax Audit Engine & CSV Export (`/admin/inventory/export`)**:
  - Automated financial audit metrics: Gross Sales Revenue (live sales from completed customer orders), Cost of Goods Sold (COGS), Operating Expenses (OpEx), Capital Asset Valuation, Net EBITDA Profit/Loss, 5% Output GST, and 25% Income Tax estimates.
  - **One-Click CA Audit CSV Export**: Download itemized audit spreadsheets for tax filings (`teachar_store_inventory_tax_audit_fy.csv`).
- **Staff Speed, SLA & Satisfaction Analytics (`/admin/staff-performance`)**:
  - Track staff preparation speed against custom estimated SLAs, on-time fulfillment rates, and customer 5-star ratings formatted in horizontal metric grids.
- **Offers & Coupon Management (`/admin/coupons`)**: Create, validate, and manage discount promo codes.
- **Staff & Admin User Management (`/admin/users`)**: Create and manage staff/admin user accounts.
- **Predefined Cancellation Reasons Management**: Add and delete system-wide cancellation reasons.
- **System Audit Logs Trail (`/admin/audit-logs`)**: Immutable audit trail logging user registrations, logins, status changes, inventory additions, and expense records.
- **Menu Item Management (`/admin/menu`)**: Add menu items with image uploads, edit details, toggle availability (*Available* / *Sold Out*), or delete items.

---

## 🔑 Demo Accounts

For testing all roles and user flows:

| Role | Email | Password | Access Level |
|---|---|---|---|
| **Superadmin** | `superadmin@teachar.in` | `SuperAdmin@123` | Full System Access, Store Inventory, Tax Audit, User Management, Audit Logs |
| **Admin** | `admin@teachar.in` | `Admin@123` | Executive Dashboard, Order Management, Menu Management, Store Inventory & Expenses |
| **Staff** | `staff@teachar.in` | `Staff@123` | Order Search, Order Claim Tagging, Custom Prep SLA Input, Status Updates, Bill Generation |
| **Client** | `client@teachar.in` | `Client@123` | Menu Browsing, Checkout, Order Tracking, Rating & Review |

---

## ⚡ High-Performance Architecture

The application uses a **Multi-File Domain-Isolated Architecture** (`MultiFileRepository`) engineered for high-concurrency (100+ concurrent user logins & order creations) using **100% native Go standard library packages**:

```text
HTTP Request
     ↓
Middleware Layer (Logging, Security Headers, Session Authentication, Role Authorization)
     ↓
Handlers Layer (Web Pages, Auth, Client Portal, Staff Order Search, Admin Portal, Store Inventory & Expenses, Payment APIs)
     ↓
Services Layer (MenuService, AuthService, OrderService, AuditService, ReportService, InventoryService, CouponService)
     ↓
Multi-File Repository Layer (MultiFileRepository with Fine-Grained RWMutexes & O(1) Memory Maps)
     ↓
Domain Storage Files (data/users.json, data/sessions.json, data/orders.json, data/menu.json, data/inventory.json, data/expenses.json, data/audit_logs.json)
```

### High-Concurrency Features:
- **Domain-Isolated JSON Files**: Isolated storage for `users.json`, `sessions.json`, `orders.json`, `menu.json`, `inventory.json`, `expenses.json`, and `audit_logs.json`.
- **Fine-Grained Domain Mutexes**: Independent RWMutex locks (`usersMu`, `sessionsMu`, `ordersMu`, `menuMu`, `inventoryMu`, `expensesMu`, `auditLogsMu`) so user logins do not block menu reads, inventory updates, or order creation.
- **Fast O(1) In-Memory Indexing**: Hash maps (`usersByEmail`, `usersByID`, `sessionsByToken`, `ordersByID`, `menuItemsByID`) for instant sub-millisecond lookups.
- **Atomic ID Generation**: Uses `sync/atomic` (`atomic.Int64`) for lock-free sequence ID increments.
- **Crash-Resistant Atomic Writes**: Atomic write-to-temp and rename (`os.WriteFile` -> `.tmp` -> `os.Rename`).
- **100% Native Standard Library**: Built strictly with standard Go packages (`sync`, `sync/atomic`, `os`, `encoding/json`, `path/filepath`, `crypto/rand`, `crypto/sha256`).

---

## 📁 File & Project Directory Structure

```text
teachar.in/
├── main.go                 # Application entry point & service initialization
├── README.MD               # Project documentation
│
├── config/                 # Environment configuration loading
│   ├── config.go
│   └── config_test.go
│
├── models/                 # Data structures (MenuItem, User, Session, Order, InventoryItem, ExpenseEntry, TaxAuditReport, AuditLog)
│   ├── menu.go
│   ├── inventory.go
│   └── store.go
│
├── repository/             # Data access layer (Interfaces, MultiFileRepository & JSONRepository)
│   ├── menu_repository.go
│   ├── inventory_repository.go
│   ├── multi_file_repository.go
│   ├── multi_file_repository_test.go
│   ├── json_repository.go
│   └── json_repository_test.go
│
├── services/               # Business logic layer
│   ├── menu_service.go
│   ├── auth_service.go
│   ├── order_service.go
│   ├── inventory_service.go
│   ├── inventory_service_test.go
│   ├── audit_service.go
│   ├── report_service.go
│   ├── coupon_service.go
│   └── menu_service_test.go
│
├── middleware/             # HTTP middleware (Logging, Security, Auth, RequireAdmin, RequireRole)
│   ├── middleware.go
│   └── middleware_test.go
│
├── handlers/               # HTTP handlers for web pages, client, auth, staff, admin, store inventory & tax audit
│   ├── handlers.go
│   ├── pages.go
│   ├── api.go
│   ├── auth.go
│   ├── client.go
│   ├── admin.go
│   ├── inventory.go
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
│   ├── admin_reports.html  # Executive sales analytics & period filters
│   └── admin_coupons.html  # Promo codes & discount coupon management
│
├── static/                 # Static Assets
│   ├── style.css           # CSS design system, horizontal card grids & print receipt styles
│   ├── app.js              # Client JS cart, payment selector & OTP modal
│   └── images/             # Food & beverage assets
│
└── data/                   # Domain-Isolated Persistent Data Directory
    ├── users.json          # Users & Staff accounts
    ├── sessions.json       # User sessions
    ├── orders.json         # Customer orders & cancellation reasons
    ├── menu.json           # Menu items
    ├── inventory.json      # Raw materials & capital asset inventory
    ├── expenses.json       # Operating expenditure vouchers
    └── audit_logs.json     # System audit trail logs
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

---

## 📜 License

Distributed under the MIT License.
