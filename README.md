# TEACHAR.in

A modern, high-performance, and responsive web application for **TEACHAR**, built using **100% Pure Go Standard Library** (zero third-party dependencies).

---

## 📸 Home Screen

![TEACHAR Home Screen](static/images/home-screenshot.png)

---

## 🌟 Overview

**TEACHAR.in** is a digital platform for an artisanal tea, coffee, and gourmet snack house. It demonstrates the power, performance, and completeness of Go's standard library for web development—featuring domain-isolated persistent JSON storage, cryptographic authentication, role-based access control, an interactive customer shopping cart, multi-channel payment gateway, real-time order tracking, live staff order search, printable GST tax invoices, customer rating & review system, and a comprehensive superadmin executive dashboard.

---

## 🚀 Comprehensive Feature List

### 🍵 Customer Features
- **Artisanal Menu (`/menu`)**: Browse categorised items (*Tea*, *Coffee*, *Snacks*, *Cold Beverages*) with live instant search and category filtering tabs.
- **Enhanced 2-Column Checkout Drawer**:
  - **Spacious Order Summary**: Clear view of items, quantities, and price breakdown.
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
- **Duplicate-Order Protection (Staff Claim Tagging)**: When a staff member picks up or updates an order, it is automatically tagged with the Staff's Name and Staff ID (`AssignedStaffID`, `AssignedStaffName`), preventing duplicate processing.
- **Mandatory Order Cancellation Reasons**: When marking an order as *Cancelled*, staff must select a reason from a predefined dropdown list (managed by Superadmin) or enter a custom reason. The cancellation note is displayed on both staff dashboards and customer receipts.
- **Printable & Downloadable GST Tax Invoices**:
  - Click **"Generate Bill"** on any order row to open a formatted **GST Tax Invoice Modal** with 5% GST breakdown, subtotal, and net payable amount.
  - **Scrollable Large Bills**: Multi-item bills feature vertical scrolling (`max-height: 88vh`).
  - **Always-Visible Sticky Footer**: Thermal print button (`window.print()`) and **Download Bill** button (`downloadBillAsPDF()`) stay locked at the bottom.

#### 👑 Superadmin & Management Portal (`/admin`)
- **Role-Based Access Control (RBAC)**: Support for `superadmin`, `admin`, `staff`, and `client` roles.
- **Grouped Navigation UX**: Organized top dropdown navigation menus (*System Management*, *Staff Control*, *Audit Logs*).
- **Staff & Admin User Management (`/admin/users`)**: Create and manage staff/admin accounts with clean list views.
- **Predefined Cancellation Reasons Management**: Add and delete system-wide cancellation reasons.
- **Audit Logs Trail (`/admin/audit-logs`)**: Complete immutable audit trail of system events, logins, and status updates.
- **Executive Dashboard (`/admin`)**: Business metrics including Total Revenue (₹), Total Orders Count, Active Orders Count, and Total Menu Items.
- **Menu Item Management (`/admin/menu`)**: Add new menu items with image uploads, edit item details, toggle availability (*Available* / *Sold Out*), or delete items.

---

## 🔑 Demo Accounts

For testing all roles and user flows:

| Role | Email | Password | Access Level |
|---|---|---|---|
| **Superadmin** | `superadmin@teachar.in` | `SuperAdmin@123` | Full System Access, User Management, Cancellation Reasons, Audit Logs |
| **Admin** | `admin@teachar.in` | `Admin@123` | Dashboard Metrics, Order Management, Menu Management |
| **Staff** | `staff@teachar.in` | `Staff@123` | Order Search, Order Claim Tagging, Status Updates, Bill Generation |
| **Client** | `client@teachar.in` | `Client@123` | Menu Browsing, Checkout, Order Tracking, Rating & Review |

---

## ⚡ High-Performance Architecture

The application uses a **Multi-File Domain-Isolated Architecture** (`MultiFileRepository`) engineered for high-concurrency (100+ concurrent user logins & order creations) using **100% native Go standard library packages**:

```text
HTTP Request
     ↓
Middleware Layer (Logging, Security Headers, Session Authentication, Role Authorization)
     ↓
Handlers Layer (Web Pages, Auth, Client Portal, Staff Order Search, Admin Portal, Payment APIs)
     ↓
Services Layer (MenuService, AuthService, OrderService, AuditService, ReportService)
     ↓
Multi-File Repository Layer (MultiFileRepository with Fine-Grained RWMutexes & O(1) Memory Maps)
     ↓
Domain Storage Files (data/users.json, data/sessions.json, data/orders.json, data/menu.json, data/audit_logs.json)
```

### High-Concurrency Features:
- **Domain-Isolated JSON Files**: Isolated storage for `users.json`, `sessions.json`, `orders.json`, `menu.json`, and `audit_logs.json`.
- **Fine-Grained Domain Mutexes**: Independent RWMutex locks (`usersMu`, `sessionsMu`, `ordersMu`, `menuMu`, `auditLogsMu`) so user logins do not block menu reads or order creation.
- **Fast O(1) In-Memory Indexing**: Hash maps (`usersByEmail`, `usersByID`, `sessionsByToken`, `ordersByID`, `menuItemsByID`) for instant sub-millisecond lookups.
- **Atomic ID Generation**: Uses `sync/atomic` (`atomic.Int64`) for lock-free sequence ID increments.
- **Crash-Resistant Atomic Writes**: Atomic write-to-temp and rename (`os.WriteFile` -> `.tmp` -> `os.Rename`).
- **100% Native Standard Library**: Built strictly with standard Go packages (`sync`, `sync/atomic`, `os`, `encoding/json`, `path/filepath`, `crypto/rand`, `crypto/sha256`).

---

## 📁 Project Structure

```text
teachar.in/
│
├── go.mod                  # Go module definition
├── main.go                 # Application entry point & server setup
├── README.MD               # Project documentation
│
├── config/                 # Environment configuration loading
│   ├── config.go
│   └── config_test.go
│
├── models/                 # Data structures (MenuItem, User, Session, Order, OrderItem, AuditLog)
│   ├── menu.go
│   └── store.go
│
├── repository/             # Data access layer (Interfaces, MultiFileRepository & JSONRepository)
│   ├── menu_repository.go
│   ├── multi_file_repository.go
│   ├── multi_file_repository_test.go
│   ├── json_repository.go
│   └── json_repository_test.go
│
├── services/               # Business logic layer
│   ├── menu_service.go
│   ├── auth_service.go
│   ├── order_service.go
│   ├── audit_service.go
│   ├── report_service.go
│   └── menu_service_test.go
│
├── middleware/             # HTTP middleware (Logging, Security, Auth, RequireAdmin, RequireRole)
│   ├── middleware.go
│   └── middleware_test.go
│
├── handlers/               # HTTP handlers for web pages, client, auth, staff & admin
│   ├── handlers.go
│   ├── pages.go
│   ├── api.go
│   ├── auth.go
│   ├── client.go
│   ├── admin.go
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
│   └── admin_orders.html   # Live order search, Claim tagging & GST Bill modal
│
├── static/                 # Static Assets
│   ├── style.css           # CSS design system & print receipt styles
│   ├── app.js              # Client JS cart, payment selector & OTP modal
│   └── images/             # High-definition food & hero images
│
└── data/                   # Domain-Isolated Data Directory
    ├── users.json          # Users & Staff accounts
    ├── sessions.json       # User sessions
    ├── orders.json         # Customer orders & cancellation reasons
    ├── menu.json           # Menu items
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
