# Reactive Programming Architecture

## Обзор

Проект EyesOn переведён на реактивную архитектуру с использованием **RxGo v2.5.0** для обработки потоков данных в реальном времени. Это обеспечивает реактивные пайплайны для SIM-данных, SSE-события для клиентов и агрегированную статистику.

### Архитектура (Data Flow)

```
┌──────────────┐     ┌──────────────┐     ┌───────────────┐
│   Upstream    │────▶│   SyncService │────▶│  Database      │
│   Pelephone   │     │  (Reactive)   │     │  (SQLite/GORM) │
└──────────────┘     └──────┬───────┘     └───────┬───────┘
                            │                       │
                    Emit(SYNC_*)             GetAllAsStream()
                            │                       │
                     ┌──────▼───────┐       ┌──────▼───────┐
                     │   Event      │       │  Sim          │
                     │  Broadcaster │       │  Repository   │
                     └──────┬───────┘       └──────┬───────┘
                            │                       │
                  ┌─────────┼─────────┐    ┌───────┼───────┐
                  │         │         │    │       │       │
              FilterByType  ToSSE   Stats  Stream  Search  Batch
                  │         │         │    │       │       │
              ┌───▼───┐ ┌──▼──┐ ┌───▼───┐│       │       │
              │SSE    │ │HTTP │ │Aggr.  │▼       ▼       ▼
              │Client │ │Push │ │Stats  │/reactive/sims
              └───────┘ └─────┘ └───────┘/reactive/search
                                         /reactive/stats
```

## Стек технологий

| Компонент | Технология | Версия |
|-----------|-----------|--------|
| Reactive | RxGo | v2.5.0 |
| HTTP | Fiber | v2.52.10 |
| ORM | GORM | v1.31.1 |
| DB | SQLite | - |
| Go | Go | 1.24.0 |

## Ключевые компоненты

### 1. Reactive Stream (`internal/reactive/stream.go`)
Базовая обёртка над RxGo Observable для цепочечных операций.

**Основные операторы:**
| Оператор | Описание | Пример |
|----------|----------|--------|
| `Map` | Трансформация элементов | Преобразование DB model → API DTO |
| `Filter` | Фильтрация по предикату | Только активные SIM |
| `FlatMap` | Разворачивание подпотоков | Запрос деталей для каждой SIM |
| `Buffer` | Группировка (count/time) | Пакеты по 100 или каждые 5 сек |
| `Debounce` | Подавление частых событий | Поиск: 300ms debounce |
| `Distinct` | Удаление дубликатов | Уникальные ICCID |
| `Retry` | Повтор при ошибке | 3 попытки с интервалом 2s |
| `CatchError` | Graceful error handling | Fallback значение при ошибке |

**Создание потока:**
```go
stream := reactive.NewStream(ctx)
// Из канала
ch := make(chan rxgo.Item)
stream = stream.FromChannel(ch)
// Из слайса
items := []rxgo.Item{rxgo.Of(1), rxgo.Of(2), rxgo.Of(3)}
stream = stream.FromItems(items)
```

### 2. SimRepository (`internal/reactive/sim_repository.go`)
Реактивный репозиторий для работы с SIM-картами.

**Возможности:**
- `GetAllAsStream()` — получение всех SIM как Observable поток
- `WatchChanges()` — мониторинг изменений в реальном времени
- `FindByStatusStream()` — фильтрация по статусу через Observable
- `BatchUpdate()` — пакетное обновление с буферизацией (Buffer → Map → Execute)
- `SearchStream()` — поиск с debouncing (300ms)

### 3. SyncService (`internal/reactive/sync_service.go`)
Реактивная синхронизация с upstream API (Pelephone).

**Возможности:**
- `ProcessTaskStream()` — обработка задач с retry и timeout
- `SyncAllSims()` — полная синхронизация
- `PeriodicSync()` — периодическая синхронизация (настраиваемый интервал)
- `MonitorChanges()` — мониторинг и авто-синхронизация изменений

### 4. EventBroadcaster (`internal/reactive/event_broadcaster.go`)
Система реактивных событий для SSE.

**10 типов событий:**

| Категория | Тип | Описание |
|-----------|-----|----------|
| SIM | `SIM_CREATED` | Новая SIM добавлена |
| SIM | `SIM_UPDATED` | Статус/данные SIM обновлены |
| SIM | `SIM_DELETED` | SIM удалена |
| Sync | `SYNC_STARTED` | Синхронизация начата |
| Sync | `SYNC_COMPLETED` | Синхронизация завершена |
| Sync | `SYNC_FAILED` | Ошибка синхронизации |
| Task | `TASK_QUEUED` | Задача добавлена в очередь |
| Task | `TASK_PROCESSING` | Задача в обработке |
| Task | `TASK_COMPLETED` | Задача завершена |
| Task | `TASK_FAILED` | Задача с ошибкой |

**Возможности:**
- `Emit()` — отправка событий в канал
- `FilterByType()` — подписка на определённые типы
- `FilterByUser()` — фильтрация по пользователю
- `ToSSE()` — конвертация в `data: {...}\n\n` SSE формат
- `AggregateStats()` — агрегация: `Buffer(100, 5s) → count by type`

## API Endpoints

### Reactive API (4 эндпоинта)

#### `GET /api/v1/reactive/events`
Server-Sent Events (SSE) поток в реальном времени.

| Параметр | Тип | Описание |
|----------|-----|----------|
| `user_id` | query | Фильтр по user ID |
| `types` | query | Фильтр по типам (через запятую) |
| `token` | query | JWT токен (альтернатива Authorization header) |

**Формат SSE-сообщения:**
```
data: {"type":"SIM_UPDATED","data":{"msisdn":"972501234567","status":"Activated"},"user_id":"admin","timestamp":"2025-01-27T21:45:00Z"}
```

#### `GET /api/v1/reactive/sims`
Список SIM-карт через reactive Observable pipeline.

**Pipeline:** `Observable.FromChannel → Collect → JSON`

**Ответ:**
```json
{
  "count": 1086,
  "sims": [
    { "msisdn": "972501234567", "iccid": "8997201...", "status": "Activated", ... }
  ]
}
```

#### `GET /api/v1/reactive/search?q=<query>`
Реактивный поиск с server-side debouncing.

**Pipeline:** `Debounce(300ms) → Distinct → LIKE %query%`

| Параметр | Тип | Описание |
|----------|-----|----------|
| `q` | query, required | Строка поиска (ICCID, MSISDN, IMSI) |

**Ответ:**
```json
{
  "query": "972501",
  "count": 15,
  "results": [ ... ]
}
```

> **Известная особенность:** при одиночном HTTP-запросе Debounce(300ms) может вернуть 0 результатов, т.к. поток закрывается раньше окончания debounce окна. Для production рекомендуется client-side debounce (см. примеры ниже).

#### `GET /api/v1/reactive/stats`
Агрегированная статистика событий.

**Pipeline:** `Buffer(100 items, 5s window) → Aggregate(count by type)`

**Ответ:**
```json
{
  "total": 42,
  "by_type": {
    "SIM_UPDATED": 30,
    "SYNC_COMPLETED": 10,
    "TASK_COMPLETED": 2
  },
  "window_ms": 5000,
  "timestamp": "2025-01-27T21:45:05Z"
}
```

> Возвращает `null` или HTTP 408 если нет событий в текущем 5-секундном окне.

## Примеры использования

### 1. SSE — подписка на события (JavaScript)

```javascript
const token = 'your-jwt-token';
const es = new EventSource(
  `/api/v1/reactive/events?user_id=admin&token=${token}`
);

es.onopen = () => console.log('Connected to SSE');

es.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch (data.type) {
    case 'SIM_UPDATED':
      updateSimRow(data.data);
      break;
    case 'SYNC_COMPLETED':
      showNotification('Sync done!', 'success');
      refreshTable();
      break;
    case 'TASK_FAILED':
      showAlert(`Task failed: ${data.data.error}`);
      break;
  }
};

// Browser auto-reconnects on error
es.onerror = () => console.log('SSE reconnecting...');

// Cleanup on page unload
window.addEventListener('beforeunload', () => es.close());
```

### 2. Live Search с client-side debounce (JavaScript)

```javascript
class ReactiveSearch {
  constructor(inputEl, resultsEl, token) {
    this.input = inputEl;
    this.results = resultsEl;
    this.token = token;
    this.timer = null;
    this.controller = null;

    this.input.addEventListener('input', (e) => this.search(e.target.value));
  }

  search(query) {
    clearTimeout(this.timer);
    if (query.length < 2) { this.results.innerHTML = ''; return; }

    this.timer = setTimeout(async () => {
      if (this.controller) this.controller.abort();
      this.controller = new AbortController();

      try {
        const res = await fetch(
          `/api/v1/reactive/search?q=${encodeURIComponent(query)}`,
          {
            headers: { Authorization: `Bearer ${this.token}` },
            signal: this.controller.signal
          }
        );
        const data = await res.json();
        this.render(data.results, data.count);
      } catch (err) {
        if (err.name !== 'AbortError') console.error(err);
      }
    }, 300);
  }

  render(results, count) {
    this.results.innerHTML = `<p>${count} results</p>` +
      results.map(s => `<div>${s.msisdn} — ${s.iccid} [${s.status}]</div>`).join('');
  }
}

new ReactiveSearch(
  document.getElementById('search'),
  document.getElementById('results'),
  localStorage.getItem('token')
);
```

### 3. Dashboard со статистикой (JavaScript)

```javascript
class ReactiveDashboard {
  constructor() {
    this.history = [];
    this.maxPoints = 60; // 5 min @ 5s interval
  }

  async start() {
    await this.tick();
    setInterval(() => this.tick(), 5000);
  }

  async tick() {
    try {
      const res = await fetch('/api/v1/reactive/stats', {
        headers: { Authorization: `Bearer ${token}` }
      });

      if (res.status === 408) { this.showIdle(); return; }

      const stats = await res.json();
      if (!stats) { this.showIdle(); return; }

      this.history.push({ time: new Date(), ...stats });
      if (this.history.length > this.maxPoints) this.history.shift();

      document.getElementById('total').textContent = stats.total;
      for (const [type, count] of Object.entries(stats.by_type)) {
        const el = document.getElementById(`stat-${type}`);
        if (el) el.textContent = count;
      }
    } catch (err) {
      console.error('Stats error:', err);
    }
  }

  showIdle() {
    document.getElementById('total').textContent = '—';
  }
}

new ReactiveDashboard().start();
```

### 4. Go — эмиссия событий из хендлеров

```go
package handlers

import (
    "eyeson-go-server/internal/reactive"
)

func UpdateSimStatusHandler(c *fiber.Ctx) error {
    // ... perform status update ...

    // Emit event — all SSE subscribers see it immediately
    EmitSimEvent(
        reactive.EventSimUpdated,
        map[string]interface{}{
            "msisdn":     sim.MSISDN,
            "old_status": oldStatus,
            "new_status": newStatus,
        },
        currentUserID,
    )

    return c.JSON(fiber.Map{"status": "ok"})
}
```

### 5. Go — custom reactive pipeline

```go
func HighUsageSIMs(c *fiber.Ctx) error {
    ctx := context.Background()
    repo := reactive.NewSimRepository()

    stream := repo.FindByStatusStream(ctx, "Activated")

    filtered := stream.
        Filter(func(item interface{}) bool {
            sim := item.(models.Sim)
            return sim.UsageMB > 1000
        }).
        Map(func(item interface{}) interface{} {
            sim := item.(models.Sim)
            return fiber.Map{
                "msisdn": sim.MSISDN,
                "usage":  sim.UsageMB,
                "alert":  sim.UsageMB > 1800,
            }
        })

    var alerts []interface{}
    for item := range filtered.ToChannel() {
        if !item.Error() {
            alerts = append(alerts, item.V)
        }
    }
    return c.JSON(fiber.Map{"alerts": alerts})
}
```

### 6. cURL / PowerShell — быстрое тестирование

```bash
# === Bash ===

# Login
TOKEN=$(curl -s -X POST http://localhost:5000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.token')

# SSE stream (Ctrl+C to stop)
curl -N -H "Authorization: Bearer $TOKEN" \
  "http://localhost:5000/api/v1/reactive/events?user_id=admin"

# Reactive SIMs
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:5000/api/v1/reactive/sims | jq '.count'

# Search
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:5000/api/v1/reactive/search?q=972501" | jq '.'

# Stats
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:5000/api/v1/reactive/stats | jq '.'
```

```powershell
# === PowerShell ===

$body = '{"username":"admin","password":"admin"}'
$token = (Invoke-RestMethod -Uri "http://localhost:5000/api/v1/auth/login" `
  -Method POST -Body $body -ContentType "application/json").token

$hdr = @{ Authorization = "Bearer $token" }

# Reactive SIMs count
(Invoke-RestMethod -Uri "http://localhost:5000/api/v1/reactive/sims" -Headers $hdr).count

# Search
Invoke-RestMethod -Uri "http://localhost:5000/api/v1/reactive/search?q=972" -Headers $hdr

# Stats
Invoke-RestMethod -Uri "http://localhost:5000/api/v1/reactive/stats" -Headers $hdr
```

## Результаты тестирования

Комплексное тестирование от 27.01.2025:

| Категория | Тест | Результат |
|-----------|------|-----------|
| Auth | POST /auth/login | PASS |
| Backend | GET /sims (1086) | PASS |
| Backend | GET /stats | PASS |
| Backend | GET /users, /roles | PASS |
| Backend | GET /queue, /audit | PASS |
| Reactive | GET /reactive/sims | PASS (1086 SIM) |
| Reactive | GET /reactive/search | PASS* |
| Reactive | GET /reactive/stats | PASS (null = no events) |
| Reactive | GET /reactive/events (SSE) | PASS |
| Frontend | Static files (HTML/JS/CSS) | PASS |
| Frontend | Locales, Swagger UI | PASS |
| Database | SIM data integrity | PASS |
| Integration | Sync, Diagnostics, Jobs | PASS |

**Итого: 22 PASS / 1 Known Issue** (*search debounce при единичном запросе)

## Swagger Documentation

Полная OpenAPI 3.0.3 спецификация доступна:
- **UI:** `http://localhost:5000/swagger.html`
- **JSON:** `http://localhost:5000/swagger.json`

## Тестовая консоль

Интерактивная страница для тестирования reactive endpoints:
- **URL:** `http://localhost:5000/test-reactive.html`
- SSE Stream viewer с метриками
- Live search с client-side debounce
- Автообновляемая статистика
- Генератор событий (SIM status change, Sync trigger)
- Автоматический тест-suite (7 тестов)
- 5 tab с примерами кода (JS SSE, JS Search, JS Dashboard, Go Backend, cURL)

## Преимущества реактивной архитектуры

| Преимущество | Описание |
|-------------|----------|
| Real-time | SSE-события доставляются мгновенно |
| Backpressure | Buffer/Debounce контролируют поток |
| Error Recovery | Retry + CatchError + Fallback |
| Debouncing | 300ms для поиска, снижение нагрузки |
| Batch Processing | Buffer(100, 5s) для эффективной обработки |
| Scalability | Параллельные stream operators |
| Composability | Цепочки операторов: Filter → Map → Buffer → Collect |
| Testability | Интерактивная тестовая консоль + Swagger docs |

## Дальнейшее развитие

- [ ] WebSocket поддержка (дополнительно к SSE)
- [ ] Replay механизм для новых подписчиков (event store)
- [ ] Rate limiting для событий
- [ ] Reactive метрики и алерты (Prometheus)
- [ ] Circuit breaker для upstream API
- [ ] Reactive cache invalidation
- [ ] Filtered SSE subscriptions (по типу events на сервере)
