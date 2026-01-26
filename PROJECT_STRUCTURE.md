# 📐 EyesOn Project Structure

> Last Updated: January 26, 2026

## Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           EYESON PROJECT                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   eyeson-go-server/          eyeson-gui/                                │
│   ┌─────────────────┐        ┌─────────────────┐                        │
│   │   Go Backend    │        │ React Frontend  │                        │
│   │   Fiber v2.52   │        │   TypeScript    │                        │
│   │   Port: 5000    │        │   Vite Build    │                        │
│   └────────┬────────┘        └────────┬────────┘                        │
│            │                          │                                  │
│            │         npm run build    │                                  │
│            │     ←─────────────────── │                                  │
│            │       (copy to static)   │                                  │
│            │                          │                                  │
│   ┌────────▼────────┐                                                   │
│   │  static/ folder │                                                   │
│   │  (serves SPA)   │                                                   │
│   └─────────────────┘                                                   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 📁 Complete Directory Structure

```
eyeson-go/
│
├── 📄 AGENT_SKILLS.md          # AI Agent knowledge & methodology
├── 📄 ARCHITECTURE.md          # System architecture documentation
├── 📄 PROJECT_STRUCTURE.md     # This file
├── 📄 README.md                # Quick start guide
│
├── 📂 eyeson-go-server/        # ══════ GO BACKEND ══════
│   │
│   ├── 📂 cmd/
│   │   └── 📂 server/
│   │       └── 📄 main.go      # Entry point
│   │                           # - Initialize config
│   │                           # - Connect database
│   │                           # - Setup routes
│   │                           # - Start Fiber server
│   │
│   ├── 📂 internal/
│   │   │
│   │   ├── 📂 config/
│   │   │   └── 📄 config.go    # Application configuration
│   │   │                       # - ServerPort (5000)
│   │   │                       # - JWTSecret
│   │   │                       # - EyesOnT credentials
│   │   │
│   │   ├── 📂 database/
│   │   │   └── 📄 db.go        # Database setup
│   │   │                       # - SQLite connection
│   │   │                       # - GORM AutoMigrate
│   │   │                       # - Seed default data
│   │   │
│   │   ├── 📂 eyesont/
│   │   │   └── 📄 client.go    # Pelephone API client
│   │   │                       # - GetProvisioningData
│   │   │                       # - UpdateProvisioningData
│   │   │                       # - GetJobList
│   │   │
│   │   ├── 📂 handlers/
│   │   │   ├── 📄 auth.go      # Authentication handlers
│   │   │   │                   # - Login
│   │   │   │                   # - GetUsers, CreateUser
│   │   │   │                   # - UpdateUser, DeleteUser
│   │   │   │                   # - ResetPassword, ChangePassword
│   │   │   │
│   │   │   ├── 📄 middleware.go # Middleware functions
│   │   │   │                   # - AuthRequired (JWT validation)
│   │   │   │                   # - RequireRole (RBAC)
│   │   │   │
│   │   │   ├── 📄 roles.go     # Role handlers
│   │   │   │                   # - GetRoles, GetRole
│   │   │   │                   # - CreateRole, UpdateRole
│   │   │   │                   # - DeleteRole
│   │   │   │
│   │   │   ├── 📄 sims.go      # SIM handlers
│   │   │   │                   # - GetSims (list, filter, sort)
│   │   │   │                   # - UpdateSim (labels)
│   │   │   │                   # - BulkChangeStatus
│   │   │   │
│   │   │   ├── 📄 jobs.go      # Job handlers
│   │   │   │                   # - GetJobs (history)
│   │   │   │
│   │   │   └── 📄 stats.go     # Statistics handlers
│   │   │                       # - GetStats (SIM counts)
│   │   │                       # - GetApiStatus (connection check)
│   │   │
│   │   ├── 📂 models/
│   │   │   ├── 📄 db.go        # GORM models
│   │   │   │                   # - User, Role, ActivityLog
│   │   │   │
│   │   │   └── 📄 api.go       # API structures
│   │   │                       # - EyesOnT request/response types
│   │   │
│   │   └── 📂 routes/
│   │       └── 📄 routes.go    # Route definitions
│   │                           # - 47 handlers total
│   │                           # - Public: login, static
│   │                           # - Protected: API routes
│   │                           # - Admin: users, roles
│   │
│   ├── 📂 static/              # ══════ STATIC FILES ══════
│   │   │
│   │   ├── 📄 index.html       # React SPA entry point
│   │   ├── 📄 swagger.html     # Swagger UI page
│   │   ├── 📄 swagger.json     # OpenAPI 3.0 specification
│   │   │
│   │   ├── 📂 assets/          # Vite build output
│   │   │   ├── 📄 index-*.js   # JavaScript bundles
│   │   │   └── 📄 index-*.css  # CSS bundles
│   │   │
│   │   └── 📂 locales/         # Internationalization
│   │       ├── 📄 en.json      # English strings
│   │       └── 📄 ru.json      # Russian strings
│   │
│   ├── 📄 eyeson.db            # SQLite database (auto-created)
│   ├── 📄 server.exe           # Compiled binary (Windows)
│   ├── 📄 go.mod               # Go module definition
│   └── 📄 go.sum               # Go dependencies lock
│
└── 📂 eyeson-gui/              # ══════ REACT FRONTEND ══════
    │
    ├── 📄 app.go               # Wails Go backend (optional)
    ├── 📄 main.go              # Wails entry point (optional)
    ├── 📄 wails.json           # Wails configuration
    │
    └── 📂 frontend/
        │
        ├── 📂 src/
        │   ├── 📄 App.tsx      # Main React component
        │   │                   # - ~2500 lines
        │   │                   # - All views in single file
        │   │                   # - State management
        │   │                   # - Theme system
        │   │
        │   ├── 📄 api.ts       # API client
        │   │                   # - TypeScript interfaces
        │   │                   # - Fetch wrappers
        │   │
        │   ├── 📄 index.css    # Styles
        │   │                   # - VS Code Dark+ theme
        │   │                   # - VS Code Light+ theme
        │   │                   # - Bootstrap overrides
        │   │
        │   └── 📄 main.tsx     # React entry point
        │
        ├── 📂 dist/            # Build output (npm run build)
        │   ├── 📄 index.html
        │   └── 📂 assets/
        │
        ├── 📄 index.html       # Development template
        ├── 📄 package.json     # NPM dependencies
        ├── 📄 tsconfig.json    # TypeScript config
        └── 📄 vite.config.ts   # Vite configuration
```

---

## 🗃️ Database Schema

### Tables

```sql
-- users table
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT,
    password_hash TEXT NOT NULL,
    role_id INTEGER REFERENCES roles(id),
    is_active BOOLEAN DEFAULT true,
    last_seen DATETIME,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

-- roles table
CREATE TABLE roles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    permissions TEXT
);

-- activity_logs table
CREATE TABLE activity_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    action TEXT NOT NULL,
    details TEXT,
    ip_address TEXT,
    created_at DATETIME
);
```

### Default Data

```yaml
roles:
  - id: 1, name: Administrator, permissions: (full access)
  - id: 2, name: Moderator, permissions: sims:read,sims:write,jobs:read
  - id: 3, name: Viewer, permissions: sims:read

users:
  - username: admin, password: admin123, role: Administrator
```

---

## 🔗 Route Map

### Public Routes (No Auth)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | /api/v1/auth/login | Login | Authenticate user |
| GET | /docs | redirect | Swagger UI |
| GET | /api/docs | redirect | Swagger UI (alt) |
| GET | /swagger.json | static | OpenAPI spec |
| GET | /* | static | React SPA |

### Protected Routes (JWT Required)

| Method | Path | Handler | Role |
|--------|------|---------|------|
| GET | /api/v1/sims | GetSims | Any |
| POST | /api/v1/sims/update | UpdateSim | Mod+ |
| POST | /api/v1/sims/bulk-status | BulkChangeStatus | Mod+ |
| GET | /api/v1/jobs | GetJobs | Any |
| GET | /api/v1/stats | GetStats | Any |
| PUT | /api/v1/auth/change-password | ChangePassword | Any |

### Admin Routes (Administrator Only)

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/users | GetUsers |
| POST | /api/v1/users | CreateUser |
| PUT | /api/v1/users/:id | UpdateUser |
| DELETE | /api/v1/users/:id | DeleteUser |
| POST | /api/v1/users/:id/reset-password | ResetPassword |
| GET | /api/v1/roles | GetRoles |
| GET | /api/v1/roles/:id | GetRole |
| POST | /api/v1/roles | CreateRole |
| PUT | /api/v1/roles/:id | UpdateRole |
| DELETE | /api/v1/roles/:id | DeleteRole |
| GET | /api/v1/api-status | GetApiStatus |

---

## 📦 Dependencies

### Go (go.mod)

```go
module eyeson-go-server

require (
    github.com/gofiber/fiber/v2 v2.52.10
    github.com/golang-jwt/jwt/v5
    golang.org/x/crypto // bcrypt
    gorm.io/gorm
    gorm.io/driver/sqlite
)
```

### React (package.json)

```json
{
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "bootstrap": "^5.3.2",
    "bootstrap-icons": "^1.11.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "typescript": "^5.0.0",
    "vite": "^4.5.0"
  }
}
```

---

## 🛠️ Build Commands

### Frontend

```powershell
cd eyeson-gui/frontend

# Development
npm run dev              # Start dev server

# Production build
npm run build            # Build to dist/

# Copy to backend
Copy-Item "dist\*" "..\..\eyeson-go-server\static\" -Recurse -Force
```

### Backend

```powershell
cd eyeson-go-server

# Build
go build -o server.exe ./cmd/server

# Run
.\server.exe             # Starts on :5000
```

### Full Rebuild

```powershell
# One-liner for full rebuild
cd eyeson-gui/frontend; npm run build; Copy-Item "dist\*" "..\..\eyeson-go-server\static\" -Recurse -Force; cd ..\..\eyeson-go-server; go build -o server.exe ./cmd/server; .\server.exe
```

---

## 📝 File Purposes Quick Reference

| File | Purpose |
|------|---------|
| `main.go` | Server entry point, startup |
| `config.go` | Configuration values |
| `db.go` (database) | DB connection, migrations |
| `db.go` (models) | GORM model definitions |
| `client.go` | Pelephone API client |
| `auth.go` | Authentication handlers |
| `middleware.go` | JWT/RBAC middleware |
| `sims.go` | SIM CRUD handlers |
| `jobs.go` | Job history handlers |
| `stats.go` | Statistics handlers |
| `roles.go` | Role CRUD handlers |
| `routes.go` | All route definitions |
| `App.tsx` | React main component |
| `api.ts` | TypeScript API client |
| `index.css` | Theme styles |
| `swagger.json` | API documentation |
| `en.json/ru.json` | Localization |
