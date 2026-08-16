# teachar.in

A modern, elegant, and minimalist web application for **TEACHAR**, built using only the Go standard library.

This project serves as the foundational digital platform for the TEACHAR brand, focusing on a clean architecture that is easy to understand, maintain, and extend. It is built without any third-party Go modules, demonstrating the power and completeness of Go's standard library for web development.

---

## Architecture

The application follows a classic layered architecture to separate concerns:

```
HTTP Request
      ↓
Router (net/http)
      ↓
Middleware (Logging, Recovery, Security)
      ↓
Handlers (Web Pages & API)
      ↓
Service Layer (Business Logic)
      ↓
Repository Layer (Data Access Abstraction)
      ↓
Data (In-Memory Store)
```

- **Handlers**: Responsible for parsing requests, calling services, and rendering responses (HTML or JSON).
- **Services**: Contain the core business logic, acting as an intermediary between handlers and data repositories.
- **Repository**: An interface-based layer that abstracts data storage. The current implementation uses an in-memory store, but it can be easily swapped with a real database (like PostgreSQL or MySQL) without changing the service or handler layers.

## Project Structure

The project is organized into packages, each with a distinct responsibility:

```text
teachar.in/
│
├── go.mod
├── main.go                 # Application entry point
│
├── config/                 # Configuration loading
├── handlers/               # HTTP handlers for pages and APIs
├── middleware/             # HTTP middleware (logging, etc.)
├── models/                 # Data structures (e.g., MenuItem)
├── repository/             # Data access layer (interfaces and implementations)
├── services/               # Business logic layer
│
├── templates/              # HTML templates
└── static/                 # Static assets (CSS, JS, images)
```

## Requirements

- Go 1.22 or newer

## Installation

No installation is required beyond having Go installed on your system. All dependencies are part of the Go standard library.

## Running Locally

1.  Clone the repository.
2.  Navigate to the project's root directory:
    ```bash
    cd teachar.in
    ```
3.  Run the application:
    ```bash
    go run .
    ```
4.  Open your web browser and visit **http://localhost:8080**.

## Building for Production

To build a single, self-contained executable:

```bash
go build -o teachar_server .
```

You can then run the application with `./teachar_server`.

## Configuration

The application is configured via environment variables:

- `APP_NAME`: The application name (default: `teachar.in`)
- `APP_HOST`: The host to bind to (default: `0.0.0.0`)
- `APP_PORT`: The port to listen on (default: `8080`)
- `APP_ENV`: The application environment (default: `development`)

## Available URLs

### Web Pages

- `GET /`: Home page
- `GET /menu`: Full menu page
- `GET /about`: About page

### API Endpoints

- `GET /health`: Basic health check
- `GET /api/status`: Detailed application status
- `GET /api/menu`: Returns the full menu in JSON format
- `GET /api/menu/{id}`: Returns a specific menu item by its ID

**Example API Call:**

```bash
curl http://localhost:8080/api/menu/1
```

## How to Extend the Application

### Adding a New Page

1.  Create a new HTML file in the `templates/` directory (e.g., `contact.html`).
2.  Add a new handler function in `handlers/pages.go`.
3.  Register the new route in `handlers/handlers.go`.

### Adding a New API Endpoint

1.  Add a new handler function in `handlers/api.go`.
2.  Register the new route in `handlers/handlers.go`, specifying the HTTP method (e.g., `POST /api/orders`).

### Adding a Menu Item

Modify the sample data in `repository/memory_menu_repository.go`.

### Future Database Integration

To switch to a real database like PostgreSQL:

1.  Create a new file, e.g., `repository/postgres_menu_repository.go`.
2.  Implement the `MenuRepository` interface in a new struct that uses a `*sql.DB` connection pool.
3.  In `main.go`, initialize the new PostgreSQL repository instead of the `MemoryMenuRepository` and inject it into the `MenuService`.

No changes will be needed in the `handlers` or `services` layers.
