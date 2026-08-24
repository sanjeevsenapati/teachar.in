# TEACHAR.in - Project Guidelines & Architecture State

This document defines the architecture, project conventions, and current state for **Antigravity** and any future AI assistants or developers working on **TEACHAR.in**.

---

## 🏛️ Project Overview & Architecture

TEACHAR.in is a high-performance Go web application with embedded SQLite database (`modernc.org/sqlite` in WAL mode). It runs two logical servers from a single application instance:

1. **🍵 Customer Storefront (`:8080` / `https://localhost:8080`)**:
   - Customer-facing web app: Menu (`/menu`), About (`/about`), Membership (`/membership`), Auth (`/login`, `/register`), Checkout Drawer, Multi-Channel Payment Gateway, Live Order Tracking (`/orders`), Customer Reviews.
   - Admin routes are unexposed / blocked (returns 404).

2. **🔐 Staff & Admin Portal (`:8081` / `https://localhost:8081`)**:
   - Private management portal for Staff, Cafe Admins, and SuperAdmins.
   - Order Queue, Staff Claim Tagging, Menu Management, Staff Speed & SLA Analytics, Inventory Register, OPEX Log, Tax Audit Statements, User Management, Promo Coupons, API Keys (`/admin/api-keys`).

---

## 📁 Directory Structure & Key Modules

- **`main.go`**: Server entrypoint, configuration loading, logger setup, signal handling.
- **`config/`**: Configuration structures and loader (`config.json`).
- **`handlers/`**: Web & REST endpoint handlers (Customer storefront, Admin, Auth, API Key auth).
- **`middleware/`**: Rate limiter, RBAC middleware, security headers, logging middleware.
- **`models/`**: Domain structs (User, Menu Item, Order, OrderItem, Review, ApiKey, OPEX, etc.).
- **`repository/`**: Database access layer (SQLite queries using parameterized statements).
- **`services/`**: Business logic, TLS self-signed cert generator, authentication services.
- **`templates/`**: Server-rendered HTML templates.
- **`static/`**: Static assets (Images, styles, JS for the backend server).
- **`static_site/`**: Standalone static site assets (`index.html`, `style.css`).
- **`data/`**: SQLite database file (`teachar.db`) and WAL files.
- **`logs/`**: Application log output (`app.log`).
- **`server.sh`**: Management script (`./server.sh start|stop|restart|status`).

---

## 🛠️ Commands & Workflows

- **Run Server**: `./server.sh start` or `go run main.go`
- **Build Server Binary**: `go build -o teachar-server .`
- **Stop Server**: `./server.sh stop`
- **Check Server Logs**: `tail -f logs/app.log`

---

## 📐 Coding Conventions & Guidelines

1. **No External Heavy Frameworks**:
   - Pure Go standard library + `modernc.org/sqlite` (CGO-free).
   - Use Vanilla CSS (`style.css`). Do NOT introduce Tailwind or Bootstrap unless explicitly requested by the user.

2. **Database Security & SQL Integrity**:
   - Always use parameterized queries in `repository/` (`?` placeholders for SQLite) to prevent SQL injection.
   - Do NOT execute raw unescaped string formatting in SQL queries.

3. **API & Error Handling Standards**:
   - JSON endpoints MUST respond with clean JSON objects (e.g. `{"success": true, "data": ...}` or `{"success": false, "error": "description"}`).
   - HTTP status codes must match RFC standards (`400 Bad Request`, `401 Unauthorized`, `403 Forbidden`, `404 Not Found`, `429 Too Many Requests`).

4. **Security & RBAC Rules**:
   - `/admin` routes must always enforce RBAC session/cookie authorization (`superadmin`, `admin`, `staff`).
   - API endpoints (`/api/...`) accept `X-API-Key: tch_live_...` headers or Bearer tokens. Secret keys are SHA-256 hashed in DB (`api_keys` table).

---

## 📝 State Maintenance Procedure for Antigravity

When performing changes on this project:
1. Check `AGENTS.md` and `README.md` to ensure architectural alignment.
2. Maintain existing directory boundaries (`handlers/`, `repository/`, `services/`, `models/`).
3. Whenever a significant change or new feature is completed, update the changelog section below or `README.md` to reflect the updated project state.

---

## 📊 Current Project Status & Roadmap

### 🟢 Completed Features
- Dual-port architecture (`:8080` storefront, `:8081` admin portal).
- Embedded CGO-free SQLite database with WAL mode.
- 2-Column checkout drawer with instant UPI, Card, NetBanking, COD payment options.
- Live order tracking with estimated preparation time & staff claim tagging.
- Automated self-signed TLS cert engine & sliding window rate limiter.
- API Key system with SHA-256 hashing (`/admin/api-keys`).

### 🟡 Active Development / In Progress
- Static site design & interactive enhancements (`static_site/index.html`, `static_site/style.css`).

### 🔴 Planned Future Enhancements
- WhatsApp / SMS customer notification integration for order status updates.
- Automated daily DB backup script (`data_backup_json/`).
- Export Tax Audit Statements & OPEX reports to CSV/Excel.
