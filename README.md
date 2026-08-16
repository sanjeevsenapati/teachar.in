# TEACHAR.in

A modern, elegant, and responsive web application for **TEACHAR**, built using **100% Pure Go Standard Library** (zero third-party dependencies).

---

## 📸 Home Screen

![TEACHAR Home Screen](static/images/home-screenshot.png)

---

## 🌟 Overview

**TEACHAR.in** is a digital platform for an artisanal tea, coffee, and snack house. It demonstrates the power, performance, and completeness of Go's standard library for web development—featuring persistent database storage, cryptographic authentication, an interactive customer shopping cart, multi-channel payment gateway, order tracking, and a comprehensive admin management dashboard.

---

## 🚀 Key Features

### 🍵 Customer Features
- **Artisanal Menu**: Browse categorised items (*Tea*, *Coffee*, *Snacks*, *Cold Beverages*) with live instant search and category filtering tabs.
- **Interactive Cart & Multi-Channel Payment Gateway**:
  - **UPI / QR Instant Pay**: Dynamic QR code preview (`teachar@upi`), auto-generated VPA string, and instant authorization.
  - **Credit / Debit Card**: Formatted card inputs with CVV validation and an interactive **3D Secure OTP Modal** verification!
  - **NetBanking**: Major Indian banks (HDFC, ICICI, SBI, Axis, Kotak).
  - **Cash on Delivery (COD)**: Pay at table or on delivery (Payment Status: *Pending COD*).
  - **Tax Calculation**: Automated 5% GST tax breakdown.
- **Customer Authentication**: Secure sign-up (`/register`), sign-in (`/login`), and sign-out (`/logout`).
- **Order & Receipt Tracking (`/orders`)**: Monitor order fulfillment status (*Pending*, *Preparing*, *Ready*, *Completed*, *Cancelled*), view Payment Method pills (*UPI*, *Card*, *COD*), Payment Status badges (*Paid*, *Pending COD*), and unique Transaction IDs (`TXN...`).
- **User Profile (`/account`)**: Manage personal account information and view purchase statistics.

### 🛡️ Admin Portal (`/admin`)
- **Executive Dashboard**: Key business metrics including Total Revenue (₹), Total Orders Count, Active Orders Count, and Total Menu Items.
- **Order & Payment Fulfillment**: View customer transaction IDs, payment methods, and update order fulfillment status dynamically in real-time.
- **Menu Management (`/admin/menu`)**: Add new menu items with image uploads, edit item details, toggle availability (*Available* / *Sold Out*), or delete items.

---

## 🔑 Demo Accounts

For testing both customer and admin experiences:

| Role | Email | Password |
|---|---|---|
| **Admin** | `admin@teachar.in` | `Admin@123` |
| **Client** | `client@teachar.in` | `Client@123` |

---

## 🏗️ Architecture

The application follows a clean layered architecture with strict separation of concerns:

```text
HTTP Request
     ↓
Middleware (Logging, Security Headers, Session Authentication, Role Authorization)
     ↓
Handlers Layer (Web Pages, Auth, Client Portal, Admin Portal, Payment APIs)
     ↓
Services Layer (MenuService, AuthService, OrderService with Payment Verification)
     ↓
Repository Layer (JSONRepository using sync.RWMutex & atomic file operations)
     ↓
Database (data/db.json)
```

### 100% Standard Library Design
- **HTTP Routing & Server**: `net/http` with Go 1.22+ pattern matching.
- **Data Persistence**: Standard library `os`, `encoding/json`, `sync.RWMutex`, and atomic temporary file renames.
- **Security & Hashing**: `crypto/sha256` with 16-byte random salts (`crypto/rand`) and 32-byte hex session tokens stored in HTTP-only cookies.

---

## 📁 Project Structure

```text
teachar.in/
│
├── go.mod                  # Go module definition
├── main.go                 # Application entry point & server setup
├── README.MD               # Documentation & screenshot
│
├── config/                 # Environment configuration loading
│   ├── config.go
│   └── config_test.go
│
├── models/                 # Data structures (MenuItem, User, Session, Order, OrderItem)
│   ├── menu.go
│   └── store.go
│
├── repository/             # Data access layer (Interfaces & JSONRepository)
│   ├── menu_repository.go
│   ├── json_repository.go
│   └── json_repository_test.go
│
├── services/               # Business logic layer
│   ├── menu_service.go
│   ├── auth_service.go
│   ├── order_service.go
│   └── menu_service_test.go
│
├── middleware/             # HTTP middleware (Logging, Security, Auth, RequireAdmin)
│   ├── middleware.go
│   └── middleware_test.go
│
├── handlers/               # HTTP handlers for web pages, client, auth & admin
│   ├── handlers.go
│   ├── pages.go
│   ├── api.go
│   ├── auth.go
│   ├── client.go
│   ├── admin.go
│   └── handlers_test.go
│
├── templates/              # HTML5 Semantic Templates
│   ├── layout.html
│   ├── home.html
│   ├── menu.html
│   ├── about.html
│   ├── error.html
│   ├── login.html
│   ├── register.html
│   ├── client_orders.html
│   ├── client_account.html
│   ├── admin_dashboard.html
│   ├── admin_menu.html
│   └── admin_orders.html
│
├── static/                 # Static Assets
│   ├── style.css           # Custom CSS Design System with Payment Gateway UI
│   ├── app.js              # Client JS Cart, Payment Selector & OTP Modal
│   └── images/             # High-definition food & hero images
│
└── data/                   # Persistent Database directory
    └── db.json             # JSON Database file
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

## 🧪 Running Tests

To run the entire automated unit test suite:

```bash
go test -v ./...
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

## 🔗 Available Endpoints & Routes

### Web Pages
- `GET /` - Home Page
- `GET /menu` - Full Artisanal Menu Page
- `GET /about` - About Us & Location Details Page
- `GET /login` - Sign In Page
- `GET /register` - Account Registration Page

### Customer Portal
- `GET /orders` - Customer Order History & Payment Status
- `GET /account` - User Account Profile

### Admin Portal
- `GET /admin` - Executive Dashboard & Metrics
- `GET /admin/menu` - Menu Item Management Table
- `POST /admin/menu/add` - Add New Menu Item
- `POST /admin/menu/edit` - Edit Menu Item
- `POST /admin/menu/delete` - Delete Menu Item
- `POST /admin/menu/toggle` - Toggle Availability Status
- `GET /admin/orders` - Order Fulfillment & Payment List
- `POST /admin/orders/status` - Update Order Status

### API Endpoints
- `GET /health` - Health Check (`{"status":"UP"}`)
- `GET /api/status` - Detailed API status
- `GET /api/menu` - Returns full menu grouped by category (JSON)
- `GET /api/menu/{id}` - Returns specific menu item by ID (JSON)
- `POST /api/orders` - Submit new order with Payment Gateway details (JSON)

---

## 📜 License

Distributed under the MIT License.
