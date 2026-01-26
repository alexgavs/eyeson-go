# EyesOn - Architecture & Agent Guidelines

## 📋 Обзор проекта

**EyesOn** - система управления SIM-картами с веб-интерфейсом, построенная на Go (backend) и React/TypeScript (frontend).

---

## 🏗️ Структура проекта

```
eyesOn/
├── eyeson-go-server/           # Go Backend Server
│   ├── cmd/
│   │   └── server/
│   │       └── main.go         # Entry point
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go       # Конфигурация приложения
│   │   ├── database/
│   │   │   └── db.go           # SQLite + GORM, seed данные
│   │   ├── handlers/
│   │   │   ├── auth.go         # Логин, пользователи, пароли
│   │   │   ├── middleware.go   # JWT, RBAC middleware
│   │   │   ├── roles.go        # CRUD для ролей
│   │   │   ├── sims.go         # Работа с SIM-картами
│   │   │   ├── jobs.go         # Provisioning jobs
│   │   │   └── stats.go        # Статистика
│   │   ├── models/
│   │   │   ├── db.go           # GORM модели (User, Role, ActivityLog)
│   │   │   └── api.go          # API структуры для EyesOnT
│   │   ├── routes/
│   │   │   └── routes.go       # Fiber роуты
│   │   └── services/
│   │       └── eyesont.go      # Клиент EyesOnT API
│   ├── static/                 # Статические файлы (React build)
│   │   ├── index.html          # React SPA entry
│   │   ├── login.html          # Страница логина
│   │   └── assets/             # JS/CSS бандлы
│   └── eyeson.db               # SQLite база данных
│
├── eyeson-gui/                 # React Frontend
│   └── frontend/
│       ├── src/
│       │   ├── App.tsx         # Главный компонент (~2000 строк)
│       │   └── api.ts          # API клиент
│       ├── dist/               # Production build
│       └── package.json
│
├── dashboard/                  # Legacy Flask Dashboard (не используется)
└── tests/                      # Python тесты
```

---

## 🔧 Технологический стек

### Backend (Go)
| Компонент | Технология | Версия |
|-----------|------------|--------|
| Web Framework | Fiber | v2.52.10 |
| ORM | GORM | latest |
| Database | SQLite | embedded |
| Auth | JWT | golang-jwt/v5 |
| Password | bcrypt | golang.org/x/crypto |

### Frontend (React)
| Компонент | Технология | Версия |
|-----------|------------|--------|
| Framework | React | 18.x |
| Language | TypeScript | 5.x |
| Build Tool | Vite | 4.5.x |
| UI | Bootstrap | 5.3.2 |

---

## 📊 Модели данных

### User
```go
type User struct {
    gorm.Model
    Username     string    // Уникальное имя
    Email        string    // Email пользователя
    PasswordHash string    // bcrypt hash
    RoleID       uint      // FK на Role
    Role         Role      // Связь
    LastSeen     time.Time
    IsActive     bool      // default: true
}
```

### Role
```go
type Role struct {
    ID          uint
    Name        string    // Administrator, Moderator, Viewer
    Description string
    Permissions string    // Comma-separated permissions
}
```

### Роли по умолчанию
| Роль | Права |
|------|-------|
| Administrator | Полный доступ: users, roles, sims, jobs, stats |
| Moderator | sims.read, sims.write, jobs.read, stats.read |
| Viewer | sims.read, jobs.read, stats.read |

---

## 🛣️ API Endpoints

### Auth (Public)
```
POST /api/v1/auth/login          # Вход в систему
```

### Auth (Protected)
```
PUT  /api/v1/auth/change-password # Смена своего пароля
```

### Users (Admin only)
```
GET    /api/v1/users              # Список пользователей
POST   /api/v1/users              # Создание пользователя
PUT    /api/v1/users/:id          # Обновление пользователя
DELETE /api/v1/users/:id          # Удаление пользователя
POST   /api/v1/users/:id/reset-password # Сброс пароля
```

### Roles (Admin only)
```
GET    /api/v1/roles              # Список ролей
GET    /api/v1/roles/:id          # Получить роль
POST   /api/v1/roles              # Создать роль
PUT    /api/v1/roles/:id          # Обновить роль
DELETE /api/v1/roles/:id          # Удалить роль
```

### SIMs (Protected)
```
GET  /api/v1/sims                 # Список SIM с фильтрами
POST /api/v1/sims/update          # Обновить SIM (Admin/Moderator)
POST /api/v1/sims/bulk-status     # Массовая смена статуса
```

### Jobs & Stats (Protected)
```
GET /api/v1/jobs                  # Provisioning jobs
GET /api/v1/stats                 # Статистика
```

---

## 🎨 Frontend Architecture

### Навигация (NavPage)
```typescript
type NavPage = 'sims' | 'jobs' | 'stats' | 'admin' | 'profile';
```

### Компоненты UI
| Tab | Описание |
|-----|----------|
| 📱 SIM Cards | Таблица SIM с сортировкой, фильтрами, пагинацией |
| 📋 Jobs | Provisioning jobs с фильтрами |
| 📊 Statistics | Дашборд статистики |
| ⚙️ Admin | User Management, System Settings |
| 👤 Profile | Профиль пользователя, настройки |

### State Management
- `useState` для локального состояния
- `useMemo` для вычисляемых значений (stats)
- `useCallback` для мемоизации функций
- `localStorage` + Cookies для сохранения настроек колонок

---

## 🔐 Аутентификация

### JWT Flow
1. `POST /api/v1/auth/login` → получение токена
2. Токен сохраняется в `localStorage`
3. Все запросы с `Authorization: Bearer <token>`
4. Middleware проверяет токен и роль

### Middleware Chain
```
JWTMiddleware → RequireAnyRole("Administrator") → Handler
```

---

## 🚀 Команды для разработки

### Запуск Go Server
```bash
cd eyeson-go-server
$env:PORT = "3000"
go run cmd/server/main.go
```

### Сборка React
```bash
cd eyeson-gui/frontend
npm install
npm run build
```

### Деплой React в Go Server
```powershell
Copy-Item -Path "eyeson-gui/frontend/dist/index.html" -Destination "eyeson-go-server/static/index.html" -Force
Copy-Item -Path "eyeson-gui/frontend/dist/assets/*" -Destination "eyeson-go-server/static/assets/" -Force
```

---

## 🤖 Guidelines для AI-агентов

### При работе с Backend (Go)

1. **Модели** находятся в `internal/models/db.go`
2. **Handlers** в `internal/handlers/` - один файл на домен
3. **Routes** в `internal/routes/routes.go` - группировка по middleware
4. **Формат ответов API**:
   ```json
   // Успех со списком
   {"data": [...]}
   
   // Успех с сообщением
   {"message": "Success"}
   
   // Ошибка
   {"error": "Error description"}
   ```

### При работе с Frontend (React)

1. **Главный файл**: `eyeson-gui/frontend/src/App.tsx`
2. **API клиент**: `eyeson-gui/frontend/src/api.ts`
3. **Типы** определяются в начале `App.tsx` и в `api.ts`
4. **После изменений**:
   ```bash
   npm run build
   # Скопировать dist в static
   ```

### Общие правила

1. **Порт по умолчанию**: 3000 (через `$env:PORT`)
2. **База данных**: `eyeson.db` в корне `eyeson-go-server`
3. **Удаление БД** пересоздаёт seed данные (admin/admin)
4. **Пароли**: минимум 6 символов, bcrypt хеширование

### API Request/Response форматы

#### Создание пользователя
```json
// Request POST /api/v1/users
{
  "username": "newuser",
  "email": "user@example.com",
  "password": "password123",
  "role": "Viewer"
}

// Response
{
  "message": "User created successfully",
  "user_id": 2
}
```

#### Получение пользователей
```json
// Response GET /api/v1/users
{
  "data": [
    {
      "id": 1,
      "username": "admin",
      "email": "admin@eyeson.local",
      "role": "Administrator",
      "is_active": true,
      "created_at": "2026-01-26T14:28:31Z",
      "updated_at": "2026-01-26T14:28:31Z"
    }
  ]
}
```

#### Сброс пароля
```json
// Request POST /api/v1/users/:id/reset-password
{
  "new_password": "newpassword123"
}

// Response
{
  "message": "Password reset successfully"
}
```

---

## 📝 Checklist для изменений

### Backend изменения
- [ ] Обновить модель в `models/db.go`
- [ ] Обновить handler в `handlers/`
- [ ] Обновить routes если новый endpoint
- [ ] Удалить `eyeson.db` если изменилась схема
- [ ] Перезапустить сервер

### Frontend изменения
- [ ] Обновить типы в `api.ts`
- [ ] Обновить компоненты в `App.tsx`
- [ ] Запустить `npm run build`
- [ ] Скопировать `dist/` в `static/`
- [ ] Обновить `index.html` в `static/`

---

## 🔗 Внешние зависимости

### EyesOnT API
- **URL**: `https://eot-portal.pelephone.co.il:8888`
- **Endpoints**:
  - `/ipa/apis/json/provisioning/getProvisioningData` - получение SIM
  - `/ipa/apis/json/provisioning/getProvisioningJobList` - jobs
  - `/ipa/apis/json/provisioning/UpdateStatusService` - смена статуса

---

## 📅 История изменений

### 2026-01-26
- ✅ Добавлены табы: Statistics, Admin, Profile
- ✅ Реализован User Management (CRUD)
- ✅ Добавлено поле Email в модель User
- ✅ Исправлен формат API ответов для совместимости с фронтендом
- ✅ Добавлена роль Viewer
- ✅ Удалён неиспользуемый main.html
