# 📐 EyesOn Project Structure

## Обзор архитектуры

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLIENT (Browser)                        │
│                    React SPA + Bootstrap 5                      │
└───────────────────────────────┬─────────────────────────────────┘
                                │ HTTP/REST
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      GO FIBER SERVER (:3000)                    │
│  ┌───────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │  Routes   │→ │Middleware│→ │ Handlers │→ │   Database    │  │
│  │ (routes/) │  │  (JWT)   │  │(handlers)│  │    (GORM)     │  │
│  └───────────┘  └──────────┘  └──────────┘  └───────────────┘  │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      SQLite (eyeson.db)                         │
│   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────┐   │
│   │  Users  │  │  Roles  │  │SIM Cards│  │ Activity Logs   │   │
│   └─────────┘  └─────────┘  └─────────┘  └─────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📁 Структура директорий

### Go Backend (`eyeson-go-server/`)

```
eyeson-go-server/
├── cmd/
│   └── server/
│       └── main.go           # Entry point, запуск сервера
│
├── internal/
│   ├── database/
│   │   └── db.go             # Подключение к SQLite, seed данные
│   │
│   ├── handlers/
│   │   ├── auth.go           # Login, Logout, User CRUD
│   │   ├── dashboard.go      # Dashboard статистика
│   │   ├── jobs.go           # Jobs API
│   │   ├── middleware.go     # JWT validation, Role check
│   │   ├── roles.go          # Roles API
│   │   ├── sims.go           # SIM Cards API
│   │   └── (other handlers)
│   │
│   ├── models/
│   │   └── db.go             # GORM модели (User, Role, SIM, etc.)
│   │
│   └── routes/
│       └── routes.go         # Все маршруты API
│
├── static/                   # Frontend build (копируется из dist/)
│   ├── index.html
│   ├── assets/
│   │   ├── index-*.js
│   │   └── index-*.css
│   └── (другие файлы)
│
├── eyeson.db                 # SQLite база (создаётся автоматически)
├── go.mod
└── go.sum
```

### React Frontend (`eyeson-gui/frontend/`)

```
eyeson-gui/frontend/
├── src/
│   ├── App.tsx               # Главный компонент (~2000 строк)
│   ├── api.ts                # API функции и типы
│   ├── main.tsx              # Entry point
│   └── App.css               # Стили
│
├── public/
│   └── (статические файлы)
│
├── dist/                     # Build output (npm run build)
│
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

---

## 🗃️ Модели данных

### User
```go
type User struct {
    ID        uint      `gorm:"primaryKey"`
    Username  string    `gorm:"unique;not null"`
    Email     string    `gorm:"unique"`
    Password  string    `gorm:"not null"` // bcrypt hash
    RoleID    uint      `gorm:"not null"`
    Role      Role      `gorm:"foreignKey:RoleID"`
    IsActive  bool      `gorm:"default:true"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Role
```go
type Role struct {
    ID          uint   `gorm:"primaryKey"`
    Name        string `gorm:"unique;not null"`
    Description string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Seed данные:
// 1. Administrator - полный доступ
// 2. Moderator - редактирование
// 3. Viewer - только чтение
```

### SIM Card
```go
type SIMCard struct {
    ID        uint      `gorm:"primaryKey"`
    ICCID     string    `gorm:"unique;not null"`
    MSISDN    string
    IMSI      string
    Status    string    `gorm:"default:'inactive'"`
    Provider  string
    Data      string    // JSON metadata
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Job
```go
type Job struct {
    ID        uint      `gorm:"primaryKey"`
    Name      string    `gorm:"not null"`
    Status    string    `gorm:"default:'pending'"`
    Progress  int       `gorm:"default:0"`
    Data      string    // JSON payload
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## 🔌 API Endpoints

### Публичные (без авторизации)
| Method | Endpoint | Описание |
|--------|----------|----------|
| GET | `/` | Главная страница (React SPA) |
| POST | `/api/v1/auth/login` | Авторизация |

### Защищённые (требуют JWT)
| Method | Endpoint | Roles | Описание |
|--------|----------|-------|----------|
| POST | `/api/v1/auth/logout` | Any | Выход |
| GET | `/api/v1/dashboard/summary` | Any | Статистика |
| GET | `/api/v1/sims` | Any | Список SIM |
| GET | `/api/v1/sims/:id` | Any | Детали SIM |
| POST | `/api/v1/sims` | Admin/Mod | Создать SIM |
| PUT | `/api/v1/sims/:id` | Admin/Mod | Обновить SIM |
| DELETE | `/api/v1/sims/:id` | Admin | Удалить SIM |
| GET | `/api/v1/jobs` | Any | Список Jobs |
| GET | `/api/v1/users` | Admin | Список пользователей |
| POST | `/api/v1/users` | Admin | Создать пользователя |
| PUT | `/api/v1/users/:id` | Admin | Обновить пользователя |
| DELETE | `/api/v1/users/:id` | Admin | Удалить пользователя |
| POST | `/api/v1/users/:id/reset-password` | Admin | Сброс пароля |
| GET | `/api/v1/roles` | Admin | Список ролей |

---

## 🔐 Аутентификация

### JWT Token Flow

```
1. POST /api/v1/auth/login
   Body: { "username": "admin", "password": "admin" }
   Response: { "token": "eyJhbG...", "user": {...} }

2. Сохранить token в localStorage

3. Все защищённые запросы:
   Header: Authorization: Bearer <token>

4. При ошибке 401 → перенаправить на login
```

### Token Structure
```json
{
  "user_id": 1,
  "username": "admin",
  "role": "Administrator",
  "exp": 1234567890  // 24 часа
}
```

---

## ⚛️ Frontend архитектура

### Навигация (NavPage type)
```typescript
type NavPage = 
  | 'sims'       // Список SIM карт
  | 'simDetail'  // Детали SIM
  | 'jobs'       // Список Jobs
  | 'stats'      // Статистика
  | 'admin'      // User Management
  | 'profile';   // Профиль
```

### Состояния компонента
```typescript
// Аутентификация
const [token, setToken] = useState<string | null>(...)
const [currentUser, setCurrentUser] = useState<User | null>(...)

// Навигация
const [navPage, setNavPage] = useState<NavPage>('sims')

// SIM Cards
const [sims, setSims] = useState<SimCard[]>([])
const [selectedSim, setSelectedSim] = useState<SimCard | null>(null)

// Jobs
const [jobs, setJobs] = useState<Job[]>([])

// User Management
const [users, setUsers] = useState<User[]>([])
const [roles, setRoles] = useState<Role[]>([])
const [showUserModal, setShowUserModal] = useState(false)
const [editingUser, setEditingUser] = useState<User | null>(null)
```

### Компоненты UI
```
App
├── Login Form (when !token)
└── Dashboard (when token)
    ├── Navbar
    │   └── Tabs: SIMs | Jobs | Statistics | Admin | Profile
    ├── Toast notifications
    └── Content area
        ├── SIM List
        │   ├── Search/Filters
        │   ├── Table
        │   └── Pagination
        ├── SIM Detail
        ├── Jobs List
        ├── Statistics Cards
        ├── Admin Panel (User Management)
        │   ├── User Table
        │   ├── Create/Edit Modal
        │   └── Reset Password Modal
        └── Profile
```

---

## 🔧 Конфигурация

### Go Server
```go
// Порт из переменной окружения или 3000
port := os.Getenv("PORT")
if port == "" {
    port = "3000"
}

// Статические файлы
app.Static("/", "./static")
```

### Vite
```typescript
// vite.config.ts
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:3000'  // Proxy в dev режиме
    }
  }
})
```

---

## 📊 Диаграмма зависимостей

```
┌──────────────┐     ┌──────────────┐
│   main.go    │────▶│   routes.go  │
└──────────────┘     └───────┬──────┘
                             │
        ┌────────────────────┼────────────────────┐
        ▼                    ▼                    ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ middleware.go│     │ handlers/*   │     │ database.go  │
└──────────────┘     └───────┬──────┘     └───────┬──────┘
                             │                    │
                             ▼                    ▼
                     ┌──────────────┐     ┌──────────────┐
                     │  models/db   │◀────│   eyeson.db  │
                     └──────────────┘     └──────────────┘
```

---

## 🚀 Development Workflow

```bash
# 1. Запустить Go сервер
cd eyeson-go-server
$env:PORT = "3000"
go run cmd/server/main.go

# 2. Для frontend разработки (hot reload)
cd eyeson-gui/frontend
npm run dev  # http://localhost:5173

# 3. Для production build
npm run build
Copy-Item -Path "dist/*" -Destination "../../eyeson-go-server/static/" -Recurse -Force

# 4. Открыть в браузере
http://127.0.0.1:3000
Login: admin / admin
```
