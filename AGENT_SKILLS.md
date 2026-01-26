# 🤖 AI Agent Skills & Methodology

## EyesOn Project Knowledge Base

> Last Updated: January 26, 2026

---

## 📚 Core Skills

### 1. Go Backend Development (Fiber v2.52.10)

```yaml
skill: go-fiber-backend
description: REST API development with Go Fiber framework
port: 5000
handlers: 47 registered routes

key_files:
  - eyeson-go-server/cmd/server/main.go          # Entry point
  - eyeson-go-server/internal/handlers/*.go      # API handlers
  - eyeson-go-server/internal/routes/routes.go   # Route definitions
  - eyeson-go-server/internal/models/db.go       # GORM models
  - eyeson-go-server/internal/database/db.go     # DB connection & seed
  - eyeson-go-server/internal/eyesont/client.go  # Pelephone API client

patterns:
  handler: |
    func HandlerName(c *fiber.Ctx) error {
        // Parse request
        var req RequestType
        if err := c.BodyParser(&req); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
        }
        // Business logic
        // Return response
        return c.JSON(fiber.Map{"data": result})
    }
  
  middleware: |
    func AuthRequired(c *fiber.Ctx) error {
        token := c.Get("Authorization")
        // Validate JWT
        return c.Next()
    }

  api_response:
    success: '{"result": "SUCCESS", "data": [...]}'
    error: '{"error": "Error message"}'
```

### 2. React/TypeScript Frontend

```yaml
skill: react-typescript-spa
description: Single Page Application with React 18 + TypeScript
build_tool: Vite
ui_framework: Bootstrap 5.3.2

key_files:
  - eyeson-gui/frontend/src/App.tsx      # Main component (~2500 lines)
  - eyeson-gui/frontend/src/api.ts       # API client & types
  - eyeson-gui/frontend/src/index.css    # VS Code themes (Dark+/Light+)
  - eyeson-gui/frontend/src/main.tsx     # Entry point

patterns:
  functional_component: |
    const Component: React.FC<Props> = ({ prop1, prop2 }) => {
      const [state, setState] = useState<Type>(initial);
      
      useEffect(() => {
        // Side effects
      }, [dependencies]);
      
      return <div>...</div>;
    };

  api_call: |
    const fetchData = async () => {
      try {
        const response = await fetch('/api/v1/endpoint', {
          headers: { 'Authorization': `Bearer ${token}` }
        });
        const data = await response.json();
        setState(data);
      } catch (error) {
        console.error('Error:', error);
      }
    };

  theme_system: |
    // CSS Variables based theme
    const [theme, setTheme] = useState<'dark' | 'light'>('dark');
    useEffect(() => {
      document.documentElement.setAttribute('data-theme', theme);
      localStorage.setItem('theme', theme);
    }, [theme]);
```

### 3. Database Operations (SQLite + GORM)

```yaml
skill: sqlite-gorm-database
description: SQLite database with GORM ORM
database_file: eyeson-go-server/eyeson.db

models:
  - User (id, username, email, password_hash, role_id, is_active, last_seen)
  - Role (id, name, description, permissions)
  - ActivityLog (id, user_id, action, details, ip_address, created_at)

operations:
  auto_migrate: |
    db.AutoMigrate(&User{}, &Role{}, &ActivityLog{})
  
  seed_data: |
    // Create default roles: Administrator, Moderator, Viewer
    // Create admin user: admin/admin123
  
  crud:
    create: db.Create(&model)
    read: db.First(&model, id)
    update: db.Save(&model)
    delete: db.Delete(&model, id)
    preload: db.Preload("Role").Find(&users)

important_note: |
  When changing DB schema:
  1. Delete eyeson.db
  2. Rebuild and restart server
  3. DB will be recreated with new schema
```

### 4. JWT Authentication & RBAC

```yaml
skill: jwt-rbac-auth
description: JWT tokens with Role-Based Access Control

jwt_config:
  secret: from config (JWT_SECRET env or default)
  expiration: 24 hours
  header: "Authorization: Bearer <token>"

roles:
  Administrator:
    permissions: Full access to all endpoints
    can_access: [users, roles, sims, jobs, stats, api-status]
  
  Moderator:
    permissions: SIM management and jobs
    can_access: [sims, jobs, stats]
  
  Viewer:
    permissions: Read-only access
    can_access: [sims (read), stats]

middleware_chain: |
  Routes → AuthRequired → RoleRequired("Administrator") → Handler
```

### 5. Pelephone EyesOnT API Integration

```yaml
skill: eyesont-api-client
description: Proxy to Pelephone EyesOnT API for SIM management

api_config:
  url: https://eot-portal.pelephone.co.il:8888
  user: samsonixapi
  password: (configured in config.go)

endpoints_proxied:
  - GET /api/v1/sims → getProvisioningData
  - POST /api/v1/sims/update → updateCustomerLabels
  - POST /api/v1/sims/bulk-status → changeSimStatus (bulk)
  - GET /api/v1/jobs → getProvisioningJobList
  - GET /api/v1/stats → aggregate SIM statistics

waf_notes: |
  - Pelephone has Incapsula WAF protection
  - Some requests may be blocked
  - Use limit=25+ (not limit=1) to avoid WAF triggers
```

---

## 🛠️ Development Workflow

### Build & Deploy Pipeline

```
Edit Code → Build → Copy → Restart → Test
    │           │       │        │       │
    │    [Frontend]     │        │       │
    │    npm run build  │        │       │
    │           │       │        │       │
    │           └───────┼────────┘       │
    │                   │                │
    │           Copy dist to static      │
    │                   │                │
    │    [Backend]      │                │
    │    go build       │                │
    │           │       │                │
    │           └───────┼────────────────┘
    │                   │
    └───────────────────┘
```

### Common Commands

```powershell
# Frontend build and copy
cd eyeson-gui/frontend
npm run build
Copy-Item "dist\*" "..\..\eyeson-go-server\static\" -Recurse -Force

# Backend build and run
cd eyeson-go-server
go build -o server.exe ./cmd/server
.\server.exe

# Full rebuild
Set-Location "eyeson-gui/frontend"
npm run build
Copy-Item "dist\*" "..\..\eyeson-go-server\static\" -Recurse -Force
Set-Location "..\..\eyeson-go-server"
go build -o server.exe ./cmd/server
.\server.exe
```

### Port Configuration

```yaml
default_port: 5000
configuration:
  - Environment variable: PORT=5000
  - main.go: port := os.Getenv("PORT") || "5000"
  - Config: config.ServerPort

access_urls:
  - http://localhost:5000
  - http://127.0.0.1:5000
```

---

## 🎨 Theme System (VS Code Style)

### CSS Variables Architecture

```css
/* Dark+ Theme (default) */
[data-theme="dark"] {
  --bg-primary: #1e1e1e;
  --bg-secondary: #252526;
  --bg-tertiary: #2d2d30;
  --text-primary: #cccccc;
  --text-secondary: #858585;
  --accent: #0e639c;
  --accent-hover: #1177bb;
  --border: #3c3c3c;
  --success: #4ec9b0;
  --warning: #dcdcaa;
  --error: #f14c4c;
}

/* Light+ Theme */
[data-theme="light"] {
  --bg-primary: #ffffff;
  --bg-secondary: #f3f3f3;
  --text-primary: #1e1e1e;
  --accent: #0066b8;
}
```

### Theme Toggle Implementation

```typescript
// In App.tsx
const [theme, setTheme] = useState<'dark' | 'light'>(() => {
  return (localStorage.getItem('theme') as 'dark' | 'light') || 'dark';
});

useEffect(() => {
  document.documentElement.setAttribute('data-theme', theme);
  localStorage.setItem('theme', theme);
}, [theme]);

// Toggle button
<button onClick={() => setTheme(t => t === 'dark' ? 'light' : 'dark')}>
  {theme === 'dark' ? '☀️' : '🌙'}
</button>
```

---

## 📡 API Documentation (Swagger)

### Swagger Setup

```yaml
swagger_files:
  - static/swagger.json      # OpenAPI 3.0 spec (English)
  - static/swagger.html      # Swagger UI page

access_urls:
  - /docs                    # Swagger UI
  - /api/docs                # Alternative URL
  - /swagger.json            # Raw OpenAPI spec

features:
  - Try-it-out enabled
  - JWT auth persistence
  - All endpoints documented
  - Request/response examples
```

### Localization Files

```yaml
locales_directory: static/locales/

files:
  - en.json    # English (primary)
  - ru.json    # Russian translation

structure:
  - common: App-wide strings
  - navigation: Menu items
  - auth: Login/logout messages
  - dashboard: Dashboard UI
  - sims: SIM management
  - jobs: Job tracking
  - users: User management
  - roles: Role management
  - errors: Error messages
  - api: API documentation strings
```

---

## 🔧 Troubleshooting Guide

### Common Issues

| Issue | Solution |
|-------|----------|
| Cannot GET /index.html | Check working directory. main.go must `os.Chdir(filepath.Dir(exePath))` |
| Port already in use | `Get-Process -Name "server" \| Stop-Process -Force` |
| Theme not applying | Check `data-theme` attribute on `<html>`, verify CSS variables |
| WAF blocking requests | Use `limit=25+` instead of `limit=1` |
| Modal not draggable | Add mouse event handlers (onMouseDown, onMouseMove, onMouseUp) |
| Roles not in dropdown | Add CSS: `select option { background-color: var(--bg-secondary); }` |

---

## 📊 Current Project State

### Completed Features

- ✅ Go Fiber backend server (47 handlers)
- ✅ React SPA frontend with TypeScript
- ✅ JWT authentication (24h tokens)
- ✅ Role-based access control (3 roles)
- ✅ SIM management (list, filter, sort, search)
- ✅ Bulk SIM status change (Activate/Suspend)
- ✅ Jobs history and tracking
- ✅ User management (CRUD)
- ✅ Role management (CRUD)
- ✅ VS Code themes (Dark+/Light+)
- ✅ API status check (Admin only)
- ✅ Swagger API documentation (OpenAPI 3.0)
- ✅ Localization files (EN/RU)
- ✅ GitHub repository (alexgavs/eyeson-go)

### Active Configuration

```yaml
server:
  port: 5000
  handlers: 47
  framework: Fiber v2.52.10

database:
  type: SQLite
  file: eyeson.db
  orm: GORM

frontend:
  framework: React 18
  build: Vite
  theme: VS Code Dark+ (default)

api:
  upstream: https://eot-portal.pelephone.co.il:8888
  auth: JWT Bearer
  docs: /docs (Swagger UI)

github:
  repo: alexgavs/eyeson-go
  branch: main
```

---

## 🚀 Future Improvements

```yaml
performance:
  - Redis caching layer
  - Connection pooling
  - Request rate limiting

features:
  - SIM usage charts/graphs
  - Export to CSV/Excel
  - Email notifications
  - Audit log viewer
  - Multi-language UI switcher

security:
  - Refresh token rotation
  - Password complexity rules
  - Login attempt limiting
  - IP whitelisting

devops:
  - Docker containerization
  - CI/CD pipeline
  - Automated testing
  - Health check endpoints
```

---

## 📝 Agent Session Notes

### When Continuing Work

1. **Read this file first** - Contains all patterns and solutions
2. **Check ARCHITECTURE.md** - For API reference
3. **Check PROJECT_STRUCTURE.md** - For file locations
4. **Review terminal history** - For recent context

### Key Files to Modify

| Task | Files |
|------|-------|
| Add new API endpoint | routes.go, handlers/*.go |
| Change UI component | App.tsx |
| Modify themes | index.css |
| Update API docs | swagger.json |
| Add translations | locales/*.json |
| Change DB schema | models/db.go → delete eyeson.db |

### Critical Reminders

```
⚠️ ALWAYS rebuild frontend after changes:
   npm run build && copy to static/

⚠️ ALWAYS restart server after backend changes:
   go build && ./server.exe

⚠️ DELETE eyeson.db when changing DB schema

⚠️ Use port 5000 (not 3000 or 8080 - may be blocked)

⚠️ Working directory must be eyeson-go-server/

⚠️ Pelephone API has WAF - avoid limit=1 in requests
```
