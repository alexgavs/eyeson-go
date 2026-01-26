# EyesOn - System Architecture

> Last Updated: January 26, 2026

## 📋 Overview

**EyesOn** is a SIM card management system with a web interface, built on Go (backend) and React/TypeScript (frontend). It acts as a secure proxy to the Pelephone EyesOnT API, providing authentication, caching, user management, and role-based access control.

---

## 🏗️ System Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           CLIENT (Browser)                               │
│                      React 18 SPA + Bootstrap 5                          │
│                      VS Code Dark+/Light+ Themes                         │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │ HTTP/REST
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       GO FIBER SERVER (:5000)                            │
│  ┌───────────┐  ┌──────────────┐  ┌──────────┐  ┌─────────────────────┐ │
│  │  Routes   │→ │  Middleware  │→ │ Handlers │→ │      Database       │ │
│  │ (47 total)│  │ (JWT/RBAC)   │  │  (CRUD)  │  │   (SQLite/GORM)     │ │
│  └───────────┘  └──────────────┘  └────┬─────┘  └─────────────────────┘ │
│                                        │                                 │
│                                        ▼                                 │
│                              ┌──────────────────┐                        │
│                              │  EyesOnT Client  │                        │
│                              │  (API Proxy)     │                        │
│                              └────────┬─────────┘                        │
└───────────────────────────────────────┼─────────────────────────────────┘
                                        │ HTTPS
                                        ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    PELEPHONE EyesOnT API (:8888)                         │
│              https://eot-portal.pelephone.co.il:8888                     │
│                     (External SIM Management)                            │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 📁 Project Structure

```
eyeson-go/
├── eyeson-go-server/           # Go Backend Server
│   ├── cmd/
│   │   └── server/
│   │       └── main.go         # Entry point, server startup
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go       # App configuration
│   │   ├── database/
│   │   │   └── db.go           # SQLite + GORM, seed data
│   │   ├── eyesont/
│   │   │   └── client.go       # Pelephone API client
│   │   ├── handlers/
│   │   │   ├── auth.go         # Login, users, passwords
│   │   │   ├── middleware.go   # JWT, RBAC middleware
│   │   │   ├── roles.go        # Role CRUD
│   │   │   ├── sims.go         # SIM operations
│   │   │   ├── jobs.go         # Job tracking
│   │   │   └── stats.go        # Statistics
│   │   ├── models/
│   │   │   ├── db.go           # GORM models
│   │   │   └── api.go          # API structures
│   │   └── routes/
│   │       └── routes.go       # All routes (47 handlers)
│   ├── static/                 # Frontend build + assets
│   │   ├── index.html          # React SPA entry
│   │   ├── swagger.html        # Swagger UI
│   │   ├── swagger.json        # OpenAPI 3.0 spec
│   │   ├── assets/             # JS/CSS bundles
│   │   └── locales/            # i18n files
│   │       ├── en.json
│   │       └── ru.json
│   └── eyeson.db               # SQLite database
│
├── eyeson-gui/                 # React Frontend
│   └── frontend/
│       ├── src/
│       │   ├── App.tsx         # Main component (~2500 lines)
│       │   ├── api.ts          # API client
│       │   ├── index.css       # VS Code themes
│       │   └── main.tsx        # Entry point
│       ├── dist/               # Production build
│       └── package.json
│
├── AGENT_SKILLS.md             # AI Agent knowledge base
├── ARCHITECTURE.md             # This file
├── PROJECT_STRUCTURE.md        # Detailed structure
└── README.md                   # Quick start guide
```

---

## 🔧 Technology Stack

### Backend (Go)

| Component | Technology | Version |
|-----------|------------|---------|
| Web Framework | Fiber | v2.52.10 |
| ORM | GORM | latest |
| Database | SQLite | embedded |
| Auth | JWT | golang-jwt/v5 |
| Password | bcrypt | golang.org/x/crypto |

### Frontend (React)

| Component | Technology | Version |
|-----------|------------|---------|
| Framework | React | 18.x |
| Language | TypeScript | 5.x |
| Build Tool | Vite | 4.5.x |
| UI | Bootstrap | 5.3.2 |
| Icons | Bootstrap Icons | 1.11.x |

---

## 📊 Data Models

### User

```go
type User struct {
    gorm.Model
    Username     string    `gorm:"uniqueIndex;not null"`
    Email        string
    PasswordHash string    `gorm:"not null"`
    RoleID       uint
    Role         Role      `gorm:"foreignKey:RoleID"`
    LastSeen     time.Time
    IsActive     bool      `gorm:"default:true"`
}
```

### Role

```go
type Role struct {
    ID          uint   `gorm:"primaryKey"`
    Name        string `gorm:"uniqueIndex;not null"`
    Description string
    Permissions string // Comma-separated: "sims:read,sims:write,users:read"
}
```

### Default Roles

| Role | Permissions |
|------|-------------|
| Administrator | Full access to all endpoints |
| Moderator | sims:read, sims:write, jobs:read |
| Viewer | sims:read |

---

## 🔐 Authentication & Authorization

### JWT Flow

```
1. POST /api/v1/auth/login
   Body: { username, password }
   
2. Server validates credentials
   - Check user exists
   - Compare bcrypt hash
   - Check is_active
   
3. Return JWT token (24h expiry)
   Response: { token, user }
   
4. Client includes token in all requests
   Header: Authorization: Bearer <token>
   
5. Middleware validates token
   - Parse and verify signature
   - Extract user_id, role
   - Check expiration
```

### Role-Based Access Control

```go
// Middleware chain
api := app.Group("/api/v1")
api.Use(handlers.AuthRequired)

// Admin-only routes
admin := api.Group("/")
admin.Use(handlers.RequireRole("Administrator"))
admin.Get("/users", handlers.GetUsers)
admin.Post("/users", handlers.CreateUser)
```

---

## 📡 API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/login | Authenticate user |
| PUT | /api/v1/auth/change-password | Change password |

### SIM Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/sims | List SIMs (paginated) |
| POST | /api/v1/sims/update | Update SIM labels |
| POST | /api/v1/sims/bulk-status | Bulk status change |

### Jobs

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/jobs | List provisioning jobs |

### Users (Admin)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/users | List users |
| POST | /api/v1/users | Create user |
| PUT | /api/v1/users/:id | Update user |
| DELETE | /api/v1/users/:id | Delete user |
| POST | /api/v1/users/:id/reset-password | Reset password |

### Roles (Admin)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/roles | List roles |
| POST | /api/v1/roles | Create role |
| PUT | /api/v1/roles/:id | Update role |
| DELETE | /api/v1/roles/:id | Delete role |

### Statistics

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/stats | SIM statistics |
| GET | /api/v1/api-status | API health (Admin) |

### Documentation

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /docs | Swagger UI |
| GET | /swagger.json | OpenAPI spec |

---

## 🌐 Pelephone API Integration

### Configuration

```go
type EyesOnTClient struct {
    BaseURL  string  // https://eot-portal.pelephone.co.il:8888
    Username string  // samsonixapi
    Password string  // (configured)
    Client   *http.Client
}
```

### Proxied Operations

| Local Endpoint | EyesOnT Endpoint | Description |
|----------------|------------------|-------------|
| GET /api/v1/sims | getProvisioningData | List SIM cards |
| POST /api/v1/sims/bulk-status | updateProvisioningData | Change SIM status |
| GET /api/v1/jobs | getProvisioningJobList | List jobs |

### Request/Response Format

```json
// Request to EyesOnT
{
  "username": "samsonixapi",
  "password": "***",
  "start": 0,
  "limit": 25,
  "sortBy": "CLI",
  "sortDirection": "ASC",
  "search": [
    {"fieldName": "MSISDN", "fieldValue": "972501234567"}
  ]
}

// Response from EyesOnT
{
  "result": "SUCCESS",
  "count": 50,
  "data": [
    {
      "MSISDN": "972501234567",
      "CLI": "0501234567",
      "SIM_STATUS_CHANGE": "Activated",
      ...
    }
  ]
}
```

---

## 🎨 Frontend Architecture

### Component Structure

```typescript
// App.tsx (~2500 lines)
function App() {
  // State
  const [user, setUser] = useState<User | null>(null);
  const [theme, setTheme] = useState<'dark' | 'light'>('dark');
  const [view, setView] = useState<View>('dashboard');
  const [sims, setSims] = useState<Sim[]>([]);
  
  // Views: login | dashboard | sims | jobs | users | roles | profile
  
  return (
    <div className="app">
      <Navbar />
      {view === 'dashboard' && <Dashboard />}
      {view === 'sims' && <SimList />}
      {view === 'jobs' && <JobList />}
      ...
    </div>
  );
}
```

### Theme System

```css
/* VS Code Dark+ (default) */
[data-theme="dark"] {
  --bg-primary: #1e1e1e;
  --bg-secondary: #252526;
  --text-primary: #cccccc;
  --accent: #0e639c;
}

/* VS Code Light+ */
[data-theme="light"] {
  --bg-primary: #ffffff;
  --bg-secondary: #f3f3f3;
  --text-primary: #1e1e1e;
  --accent: #0066b8;
}
```

---

## 📦 Deployment

### Build Process

```powershell
# 1. Build Frontend
cd eyeson-gui/frontend
npm run build

# 2. Copy to static
Copy-Item "dist\*" "..\..\eyeson-go-server\static\" -Recurse -Force

# 3. Build Backend
cd ..\..\eyeson-go-server
go build -o server.exe ./cmd/server

# 4. Run
.\server.exe
# Server starts on http://127.0.0.1:5000
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 5000 | Server port |
| JWT_SECRET | (hardcoded) | JWT signing key |
| DB_PATH | eyeson.db | SQLite file path |

---

## 🔒 Security Considerations

### Implemented

- ✅ JWT token authentication (24h expiry)
- ✅ bcrypt password hashing
- ✅ Role-based access control
- ✅ CORS configuration
- ✅ Input validation

### Recommendations

- ⚠️ Use environment variables for secrets
- ⚠️ Implement refresh token rotation
- ⚠️ Add rate limiting
- ⚠️ Enable HTTPS in production
- ⚠️ Implement audit logging

---

## 📈 Performance Notes

### Caching

- Statistics cached for 5 minutes
- Cache invalidated on SIM status change

### Pelephone API

- WAF may block requests with `limit=1`
- Use `limit=25+` for reliable operation
- Implement retry logic for timeouts
