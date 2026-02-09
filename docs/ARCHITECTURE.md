# EyesOn - System Architecture

> Last Updated: February 10, 2026

## 📋 Overview

**EyesOn** is a SIM card management system with a web interface, built on Go (backend) and React/TypeScript (frontend). It acts as a secure proxy to the Pelephone EyesOnT API, providing authentication, caching, user management, and role-based access control.

**Key Features:**
- 🔄 **DB-First Architecture** — Works offline, syncs when API available
- ⚡ **Priority Queue System** — User actions take precedence over background sync
- 📊 **Real-time UI Updates** — Live countdown, auto-refresh without F5
- 🎭 **Built-in Simulator** — Test without real Pelephone credentials

---

## 🏗️ Full System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         🌐 БРАУЗЕР (http://localhost:5000)                      │
│                      React 18 + TypeScript + Bootstrap 5                         │
│   ┌─────────────┐  ┌──────────────┐  ┌─────────┐  ┌─────────┐  ┌───────────┐   │
│   │ SIM Cards   │  │  Queue       │  │ Jobs    │  │ Admin   │  │ History   │   │
│   │ (CRUD)      │  │ (Countdown)  │  │ (Tasks) │  │ (Users) │  │ (Audit)   │   │
│   └──────┬──────┘  └──────┬───────┘  └────┬────┘  └────┬────┘  └─────┬─────┘   │
└──────────┼────────────────┼───────────────┼───────────┼──────────────┼─────────┘
           │                │               │           │              │
           └────────────────┴───────────────┴───────────┴──────────────┘
                                           │ REST API
                                           ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         🔧 GO FIBER SERVER (:5000)                               │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                              HANDLERS                                       │ │
│  │  sims.go    │  jobs.go    │  auth.go    │  history.go   │  stats.go        │ │
│  │  • GetSims  │  • GetJobs  │  • Login    │  • GetHistory │  • GetStats      │ │
│  │  • Update   │  • Execute  │  • JWT      │  • Audit Log  │  • Dashboard     │ │
│  │  • Bulk     │  • Queue    │  • RBAC     │               │                  │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                           │                                      │
│  ┌──────────────────┐    ┌────────────────┴───────────────┐                     │
│  │  📦 DATABASE     │◄───│         MODELS (GORM)          │                     │
│  │   (SQLite)       │    │  • SimCard    • User           │                     │
│  │                  │    │  • SyncTask   • Role           │                     │
│  │   eyeson.db      │    │  • SimHistory • ActivityLog    │                     │
│  └──────────────────┘    └────────────────────────────────┘                     │
│                                           │                                      │
│  ┌────────────────────────────────────────┴────────────────────────────────┐    │
│  │                     🔄 BACKGROUND SERVICES                               │    │
│  │  ┌─────────────────────────┐    ┌─────────────────────────────────┐     │    │
│  │  │       JOB WORKER        │    │           SYNCER                │     │    │
│  │  │   (каждую 1 секунду)    │    │     (каждые 5 минут)            │     │    │
│  │  │                         │    │                                 │     │    │
│  │  │  • Polls PENDING tasks  │    │  • Fetches ALL SIMs from API   │     │    │
│  │  │  • Executes API calls   │    │  • Compares API vs DB          │     │    │
│  │  │  • Updates DB + History │    │  • Creates/Updates SimCards    │     │    │
│  │  │  • Handles retries      │    │  • Records History changes     │     │    │
│  │  │  • Priority: HIGH ⚡    │    │  • Priority: LOW (yields) 🐢   │     │    │
│  │  └───────────┬─────────────┘    └──────────────────────┬──────────┘     │    │
│  │              │                                         │                │    │
│  │              └──────────────┬──────────────────────────┘                │    │
│  │                             ▼                                           │    │
│  │                    ┌─────────────────────────┐                          │    │
│  │                    │    EYESONT CLIENT       │                          │    │
│  │                    │  (API Proxy + Sessions) │                          │    │
│  │                    └───────────┬─────────────┘                          │    │
│  └────────────────────────────────┼────────────────────────────────────────┘    │
└───────────────────────────────────┼─────────────────────────────────────────────┘
                                    │ HTTP (cookie-based auth)
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    📡 PELEPHONE SIMULATOR (:8888)                                │
│                    (или реальный Pelephone API)                                  │
│                                                                                  │
│    ┌──────────────────┐   ┌───────────────────────────────────────────┐         │
│    │   Admin Panel    │   │            API Endpoints                  │         │
│    │   /web           │   │  • POST /ipa/apis/json/general/login      │         │
│    │                  │   │  • POST /ipa/apis/.../getProvisioningData │         │
│    │  • Generate SIMs │   │  • POST /ipa/apis/.../updateSIMStatus     │         │
│    │  • Set Mode      │   │                                           │         │
│    │  • View Stats    │   │  Modes: NORMAL / REFUSED / DOWN           │         │
│    └──────────────────┘   └───────────────────────────────────────────┘         │
│                                                                                  │
│    └─────────────────────── simulator.db (SQLite) ─────────────────────┘        │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## � Data Flow Diagrams

### 1️⃣ Change SIM Status Flow

```
┌─────────────┐    1. Click "Activate/Suspend"
│  FRONTEND   │────────────────────────────────┐
│  (React)    │                                │
└─────────────┘                                ▼
                                    ┌─────────────────────┐
                                    │  POST /api/sims/    │
                                    │  bulk-status        │
                                    └──────────┬──────────┘
                                               │
                                               ▼
                              ┌──────────────────────────────┐
                              │  handlers/sims.go            │
                              │  BulkChangeStatus()          │
                              │                              │
                              │  2. Create SyncTask          │
                              │     Type: "CHANGE_STATUS"    │
                              │     Status: "PENDING"        │
                              └──────────────┬───────────────┘
                                             │
                                             ▼
                                  ┌───────────────────────┐
                                  │  📦 DATABASE          │
                                  │  sync_tasks table     │
                                  └───────────┬───────────┘
                                              │
        ┌─────────────────────────────────────┘
        │                    (polls every 1 second)
        ▼
┌───────────────────────────────────────────────────────────────┐
│  jobs/worker.go                                               │
│  ProcessPendingTasks() → handleChangeStatus()                 │
│                                                               │
│  3. Call API: Client.BulkUpdate("SIM_STATE_CHANGE", status)   │
│                                                               │
│  4. Update Local DB: SimCard.Status = newStatus               │
│                                                               │
│  5. Sync from API: syncSimsFromAPI(msisdns) ← AUTO-SYNC!      │
│                                                               │
│  6. Create SimHistory records (audit trail)                   │
│                                                               │
│  7. Update SyncTask.Status = "COMPLETED"                      │
└───────────────────────────────────────────────────────────────┘
```

### 2️⃣ Background Sync Flow (Syncer)

```
┌─────────────────────────────────────────────────────────────┐
│  syncer/syncer.go                                           │
│  SyncFull() - runs every 5 minutes                          │
│                                                             │
│  1. Check for pending user tasks (Priority Check)           │
│     └── If pending → WAIT 2 seconds, then retry             │
│                                                             │
│  2. Fetch from API: GetSims(start, limit=500)               │
│                                                             │
│  3. For each batch:                                         │
│     ┌──────────────────────────────────────────────────┐    │
│     │  a) Compare API data vs Local DB                 │    │
│     │  b) If NEW → Create SimCard                      │    │
│     │  c) If CHANGED → Update SimCard + Create History │    │
│     │  d) Fields tracked: Status, IP, IMEI, ICCID      │    │
│     └──────────────────────────────────────────────────┘    │
│                                                             │
│  4. Continue until all SIMs processed                       │
└─────────────────────────────────────────────────────────────┘
```

### 3️⃣ JWT Authentication Flow

```
Frontend                   Backend                    Database
   │                          │                          │
   │  POST /api/auth/login    │                          │
   │  {username, password}    │                          │
   │────────────────────────>│                          │
   │                          │  Find User by username   │
   │                          │─────────────────────────>│
   │                          │  Compare bcrypt hash     │
   │                          │<─────────────────────────│
   │                          │                          │
   │  {token: "JWT...",       │                          │
   │   user: {...}}           │                          │
   │<────────────────────────│                          │
   │                          │                          │
   │  All subsequent requests │                          │
   │  Header: Authorization:  │                          │
   │  Bearer <token>          │                          │
   │────────────────────────>│  Middleware validates    │
```

---

## �📁 Project Structure

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
│   │   │   ├── middleware.go   # JWT, RBAC middleware (token query param for SSE)
│   │   │   ├── roles.go        # Role CRUD
│   │   │   ├── sims.go         # SIM operations
│   │   │   ├── jobs.go         # Job tracking
│   │   │   ├── stats.go        # Statistics
│   │   │   ├── queue.go        # Queue management endpoints
│   │   │   ├── audit.go        # Audit log endpoints
│   │   │   ├── sync.go         # Manual sync triggers
│   │   │   ├── upstream.go     # Upstream API config
│   │   │   ├── oauth.go        # Google OAuth handlers
│   │   │   ├── diagnostics.go  # API diagnostics
│   │   │   ├── history.go      # SIM history
│   │   │   └── reactive_handlers.go  # Reactive SSE, search, stats
│   │   ├── jobs/               # Background Worker (Priority)
│   │   │   └── worker.go       # Task consumer
│   │   ├── syncer/             # Data Synchronization
│   │   │   └── syncer.go       # Background data fetcher
│   │   ├── reactive/
│   │   │   ├── stream.go          # RxGo Observable wrapper (Map, Filter)
│   │   │   ├── sim_repository.go  # Reactive SIM data access
│   │   │   └── event_broadcaster.go # Fan-out SSE broadcaster
│   │   ├── models/
│   │   │   ├── db.go           # GORM models
│   │   │   ├── api.go          # API structures
│   │   │   ├── audit.go        # Audit models
│   │   │   ├── queue.go        # Queue models
│   │   │   └── settings.go     # Settings models
│   │   ├── services/
│   │   │   ├── queue_service.go  # Queue service
│   │   │   ├── queue.go        # Queue operations
│   │   │   ├── audit.go        # Audit service
│   │   │   └── upstream.go     # Upstream service
│   │   └── routes/
│   │       └── routes.go       # All route registration
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
├── eyeson-gui/                 # React Frontend (Vite SPA)
│   └── frontend/
│       ├── src/
│       │   ├── App.tsx         # Main component (reactive search, debounce)
│       │   ├── api.ts          # API client
│       │   ├── index.css       # VS Code themes
│       │   ├── main.tsx        # Entry point
│       │   ├── components/     # QueueView, SimDetailModal, StatusBadges, ToastContainer
│       │   ├── constants/      # App constants
│       │   ├── types/          # TypeScript types
│       │   └── utils/          # cookies, format, session
│       └── package.json
│
├── pelephone-simulator/        # Standalone API simulator
│   ├── main.go                 # Simulator entry point
│   └── web/static/             # Admin panel
│
├── tools/                      # Dev utilities
│   ├── authtest/               # OAuth test tool
│   ├── extract_pelephone_spec.py
│   └── generate_upstream_spec.py
│
├── docs/                       # Documentation
│   ├── ARCHITECTURE.md         # This file
│   ├── REACTIVE_ARCHITECTURE.md
│   ├── TESTING_REPORT.md
│   ├── DEVELOPMENT_RULES.md
│   └── design/                 # Design documents
│
└── README.md                   # Quick start guide
```

---

## ⚡ Background Processing & Concurrency

The system uses a **Priority-Based Concurrency Model** to ensure UI responsiveness.

1.  **Job Worker (`internal/jobs`)**:
    *   Polls the database every **1 second** for new tasks (User actions).
    *   Executes tasks (e.g., Change SIM Status) immediately.

2.  **Data Syncer (`internal/syncer`)**:
    *   Fetches large datasets (20k+ SIMs) from the external API in chunks.
    *   **Cooperative Multitasking**: Before processing each chunk (500 records), the Syncer checks for pending user tasks.
    *   **Yielding**: If a user task is pending, the Syncer **pauses/yields for 2 seconds** to allow the Worker to process the user's request, preventing "resource starvation".

---

## 🔧 Technology Stack

### Backend (Go)

| Component | Technology | Version |
|-----------|------------|---------|
| Web Framework | Fiber | v2.52.10 |
| Reactive | RxGo | v2.5.0 |
| ORM | GORM | v1.31.1 |
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

### SimCard (Primary Entity)

```go
type SimCard struct {
    gorm.Model
    MSISDN      string    `gorm:"uniqueIndex"` // Phone number
    CLI         string    `gorm:"index"`       // Caller Line ID
    IMSI        string    `gorm:"index"`       // Subscriber ID
    ICCID       string                         // SIM card ID
    IMEI        string                         // Device ID
    Status      string    `gorm:"index"`       // Activated/Suspended/Terminated
    RatePlan    string    `gorm:"index"`       // Tariff plan
    Label1-3    string                         // Custom labels
    APN         string                         // Access Point Name
    IP          string                         // Assigned IP
    UsageMB     float64                        // Monthly usage
    AllocatedMB float64                        // Monthly quota
    LastSession time.Time                      // Last connection
    InSession   bool                           // Currently connected
    LastSyncAt  time.Time `gorm:"index"`       // Last API sync
}
```

### SyncTask (Queue System)

```go
type SyncTask struct {
    ID           uint      `gorm:"primaryKey"`
    Type         string    `gorm:"index"`   // CHANGE_STATUS, UPDATE_SIM, SYNC_FULL
    Status       string    `gorm:"index"`   // PENDING, PROCESSING, COMPLETED, FAILED
    Payload      string    `gorm:"text"`    // JSON payload
    Result       string    `gorm:"text"`    // Error or result message
    TargetMSISDN string    `gorm:"index"`   // For quick lookup
    Attempt      int       `gorm:"default:0"`
    MaxAttempts  int       `gorm:"default:5"`
    NextRunAt    time.Time `gorm:"index"`   // Scheduled execution
    CreatedBy    string                     // Username
    IPAddress    string                     // Client IP
}
```

### SimHistory (Audit Trail)

```go
type SimHistory struct {
    ID        uint      `gorm:"primaryKey"`
    CreatedAt time.Time
    SimID     uint      `gorm:"index"`
    MSISDN    string    `gorm:"index"`
    Action    string    // STATUS_CHANGE, SYNC_UPDATE, CREATED
    Field     string    // Changed field name
    OldValue  string
    NewValue  string
    Source    string    // USER, SYNC, WORKER
    ChangedBy string    // Username
    TaskID    *uint     // Link to SyncTask
}
```

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

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 5000 | Server port |
| `DATABASE_PATH` | eyeson.db | SQLite file path |
| `EYESON_API_BASE_URL` | `http://127.0.0.1:8888` | API URL (simulator by default) |
| `EYESON_API_USERNAME` | admin | API login |
| `EYESON_API_PASSWORD` | admin | API password |
| `EYESON_API_DELAY_MS` | 10 | Delay between API requests |
| `JWT_SECRET` | change-me-in-prod | JWT signing key |

### Switching to Real Pelephone API

Edit `.env` file:
```env
EYESON_API_BASE_URL=https://eot-portal.pelephone.co.il:8888
EYESON_API_USERNAME=your_username
EYESON_API_PASSWORD=your_password
EYESON_API_DELAY_MS=1000
```

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

### Reactive (SSE + Search)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/reactive/events | SSE event stream (fan-out broadcaster) |
| GET | /api/v1/reactive/sims | SIM list via Observable pipeline |
| GET | /api/v1/reactive/search | Reactive search (`?q=` or `?q=field:value`) |
| GET | /api/v1/reactive/stats | Aggregated event statistics |

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

---

## 🚀 Quick Start

### Run with Simulator (Recommended for Testing)

```batch
# Terminal 1: Start Simulator
cd pelephone-simulator
run.bat
# → Simulator running on http://localhost:8888
# → Admin Panel: http://localhost:8888/web

# Terminal 2: Start Server
cd eyeson-go-server
build_and_run.bat
# → Server running on http://localhost:5000

# Open Browser
http://localhost:5000
# Login: admin / admin123
```

### Run with Real Pelephone API

1. Edit `eyeson-go-server/.env`
2. Set real API credentials
3. Run `build_and_run.bat`

---

## 📚 Related Documentation

| File | Description |
|------|-------------|
| [REACTIVE_ARCHITECTURE.md](REACTIVE_ARCHITECTURE.md) | Reactive layer (RxGo, SSE, EventBroadcaster) |
| [TESTING_REPORT.md](TESTING_REPORT.md) | Test results and verification |
| [DEVELOPMENT_RULES.md](DEVELOPMENT_RULES.md) | Development guidelines |
| [design/](design/) | Design documents (billing, subscriptions, hierarchy) |
