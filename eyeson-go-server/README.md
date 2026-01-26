# EyesOn Go Server

Высокопроизводительный backend сервер для управления SIM-картами через Pelephone EyesOnT API.

## 🚀 Технологии

- **Go 1.23+** - основной язык
- **Fiber v2.52.10** - веб-фреймворк (высокая производительность)
- **GORM** - ORM для работы с SQLite
- **JWT** - аутентификация
- **React + TypeScript** - встроенный фронтенд

## 📁 Структура проекта

```
eyeson-go-server/
├── cmd/
│   └── server/
│       └── main.go          # Точка входа
├── internal/
│   ├── database/
│   │   └── database.go      # Подключение к SQLite
│   ├── eyesont/
│   │   └── client.go        # Клиент Pelephone API
│   ├── handlers/
│   │   ├── auth.go          # Авторизация (login/logout)
│   │   ├── jobs.go          # Управление заданиями
│   │   ├── sims.go          # Управление SIM-картами
│   │   └── stats.go         # Статистика
│   └── models/
│       └── models.go        # Модели данных
├── static/                   # Собранный React фронтенд
└── eyeson-server.exe         # Скомпилированный сервер
```

## 🔧 Конфигурация

Переменные окружения (`.env`):

```env
EYESONT_USERNAME=your_api_username
EYESONT_PASSWORD=your_api_password
EYESONT_BASE_URL=https://eot-portal.pelephone.co.il:8888
JWT_SECRET=your_jwt_secret
PORT=8080
```

## 🛡️ API Endpoints

### Аутентификация

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/v1/login` | Вход в систему |
| POST | `/api/v1/logout` | Выход из системы |

### SIM-карты

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/v1/sims` | Список SIM-карт с фильтрацией |
| POST | `/api/v1/sims/update` | Обновление поля SIM |
| POST | `/api/v1/sims/bulk-status` | Массовое изменение статуса |

**Параметры GET `/api/v1/sims`:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| `start` | int | Смещение (default: 0) |
| `limit` | int | Количество записей (default: 25) |
| `search` | string | Поиск по всем полям |
| `status` | string | Фильтр по статусу: `Activated`, `Suspended`, `Terminated` |
| `sortBy` | string | Поле сортировки: `CLI`, `MSISDN`, `SIM_STATUS_CHANGE` |
| `sortDirection` | string | Направление: `ASC`, `DESC` |

### Задания (Jobs)

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/v1/jobs` | Список заданий |
| GET | `/api/v1/jobs/:id` | Детали задания |

### Статистика

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/v1/stats` | Статистика SIM-карт |

## ⚡ Особенности реализации

### 1. Rate Limiting для Pelephone API

Pelephone использует WAF (Incapsula), который блокирует частые запросы. Реализована глобальная защита:

```go
var (
    apiRateMutex sync.Mutex
    lastApiCall  time.Time
)

// Минимальный интервал между запросами: 1 секунда
func rateLimitedRequest() {
    apiRateMutex.Lock()
    defer apiRateMutex.Unlock()
    
    elapsed := time.Since(lastApiCall)
    if elapsed < time.Second {
        time.Sleep(time.Second - elapsed)
    }
    lastApiCall = time.Now()
}
```

### 2. Умный поиск (Smart Search)

Автоматическое определение типа поиска по паттерну:

| Паттерн | Поле | Пример |
|---------|------|--------|
| Начинается с `05` | CLI | 0502680716 |
| Начинается с `972` | MSISDN | 972502680716 |
| 15 цифр | IMSI | 425030008946193 |
| Остальное | Локальный поиск | любой текст |

### 3. Локальная фильтрация

Для полей, не поддерживаемых API (Label, RatePlan и др.), выполняется:
1. Загрузка до 5000 записей с сервера
2. Фильтрация на стороне Go сервера
3. Пагинация отфильтрованных результатов

### 4. Серверный фильтр по статусу

```go
// Фильтрация по статусу SIM-карты
statusFilter := c.Query("status", "")
if statusFilter != "" {
    // Загружаем больше данных для фильтрации
    fetchLimit = 5000
    fetchStart = 0
}

// Применяем фильтр
if strings.EqualFold(sim.SimStatusChange, statusFilter) {
    filteredData = append(filteredData, sim)
}
```

### 5. Клиентский расчёт статистики

Для избежания блокировки WAF, статистика рассчитывается на фронтенде из загруженных данных через `useMemo`:

```typescript
const stats = useMemo(() => {
  if (!allSimsData.length) return null;
  return {
    total: allSimsData.length,
    activated: allSimsData.filter(s => s.SIM_STATUS_CHANGE === 'Activated').length,
    suspended: allSimsData.filter(s => s.SIM_STATUS_CHANGE === 'Suspended').length,
    terminated: allSimsData.filter(s => s.SIM_STATUS_CHANGE === 'Terminated').length
  };
}, [allSimsData]);
```

## 🎨 Фронтенд функционал

### Управление колонками
- **Drag & Drop** для изменения порядка колонок
- **Контекстное меню** для показа/скрытия колонок
- **Cookie Storage** для сохранения настроек между сессиями

### Фильтры и поиск
- Текстовый поиск по всем полям
- Выпадающий фильтр по статусу с иконками:
  - 🟢 Activated
  - 🟡 Suspended
  - 🔴 Terminated

### Массовые операции
- Выбор нескольких SIM через чекбоксы
- Массовое изменение статуса (Activate/Suspend/Terminate)

### Pending статусы
- Отслеживание операций в процессе
- Визуальная индикация с анимацией

## 🏃 Запуск

### Разработка

```bash
# Сборка
cd eyeson-go-server
go build -o eyeson-server.exe ./cmd/server

# Запуск
./eyeson-server.exe
```

### Сборка фронтенда

```bash
cd ../eyeson-gui/frontend
npm run build

# Копирование в static
cp -r dist/* ../eyeson-go-server/static/
```

### Production

```bash
# Один исполняемый файл со встроенным фронтендом
./eyeson-server.exe
# Доступен на http://localhost:8080
```

## 📊 Модели данных

### SimData

```go
type SimData struct {
    CLI             string `json:"CLI"`
    MSISDN          string `json:"MSISDN"`
    SimStatusChange string `json:"SIM_STATUS_CHANGE"`
    CustomerLabel1  string `json:"CUSTOMER_LABEL_1"`
    CustomerLabel2  string `json:"CUSTOMER_LABEL_2"`
    CustomerLabel3  string `json:"CUSTOMER_LABEL_3"`
    IMSI            string `json:"IMSI"`
    IMEI            string `json:"IMEI"`
    RatePlanFullName string `json:"RATE_PLAN_FULL_NAME"`
    LastSessionTime  string `json:"LAST_SESSION_TIME"`
    ApnName         string `json:"APN_NAME"`
    Ip1             string `json:"IP_1"`
    // ... и другие поля
}
```

### Job

```go
type Job struct {
    JobID       int      `json:"jobId"`
    Status      string   `json:"status"`
    RequestTime int64    `json:"requestTime"`
    Actions     []Action `json:"actions"`
}

type Action struct {
    NeID          string `json:"neId"`
    RequestType   string `json:"requestType"`
    Status        string `json:"status"`
    InitialValue  string `json:"initialValue"`
    TargetValue   string `json:"targetValue"`
    ErrorDesc     string `json:"errorDesc"`
}
```

## 🔒 Безопасность

- JWT токены для авторизации
- Защита от SQL-инъекций через GORM
- Rate limiting для защиты внешнего API
- Валидация всех входных параметров

## 📝 История изменений

### v1.0.0 (Январь 2026)
- ✅ Миграция с Python/Flask на Go/Fiber
- ✅ Интеграция с Pelephone EyesOnT API
- ✅ Умный поиск по паттернам
- ✅ Серверный фильтр по статусу
- ✅ Rate limiting для WAF protection
- ✅ Клиентский расчёт статистики
- ✅ Drag & Drop колонок с Cookie storage
- ✅ Массовые операции со статусами
- ✅ Pending статусы с отслеживанием Jobs

## 📄 Лицензия

Proprietary - Samsonix Ltd.
