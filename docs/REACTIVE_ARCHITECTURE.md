# Reactive Programming Architecture

## Обзор

Проект EyesOn использует реактивную архитектуру на базе **RxGo v2.5.0** для обработки потоков данных в реальном времени. Реактивный слой обеспечивает SSE-события для клиентов, потоковый доступ к SIM-данным и агрегированную статистику.

### Архитектура (Data Flow)

```
┌──────────────┐                        ┌───────────────┐
│   Upstream    │ ─── syncer.go ──────▶ │  Database      │
│   Pelephone   │                       │  (SQLite/GORM) │
└──────────────┘                        └───────┬───────┘
                                                │
          Emit(events)                   GetAllAsStream()
          from handlers                         │
                │                        ┌──────▼───────┐
         ┌──────▼───────┐               │  Sim          │
         │   Event      │               │  Repository   │
         │  Broadcaster │               └──────┬───────┘
         └──────┬───────┘                      │
                │                         Stream → Collect
      ┌─────────┼─────────┐                   │
      │         │         │                   ▼
  Subscribe  ToSSE    Stats         /reactive/sims
      │         │         │         /reactive/search
  ┌───▼───┐ ┌──▼──┐ ┌───▼───┐
  │SSE    │ │HTTP │ │Aggr.  │
  │Client │ │Push │ │Stats  │
  └───────┘ └─────┘ └───────┘
```

## Стек технологий

| Компонент | Технология | Версия |
|-----------|-----------|--------|
| Reactive | RxGo | v2.5.0 |
| HTTP | Fiber | v2.52.10 |
| ORM | GORM | v1.31.1 |
| DB | SQLite | — |
| Go | Go | 1.24.0 |

## Ключевые компоненты

### 1. Reactive Stream (`internal/reactive/stream.go`)

Базовая обёртка над RxGo Observable для цепочечных операций.

**Операторы:**

| Оператор | Описание | Пример |
|----------|----------|--------|
| `Map` | Трансформация элементов | Преобразование DB model → API DTO |
| `Filter` | Фильтрация по предикату | Только активные SIM |

**Вспомогательные методы:**

| Метод | Описание |
|-------|----------|
| `Subscribe` | Подписка на все элементы потока |
| `ToChannel` | Конвертация Observable в канал Go |
| `Observable` | Доступ к нижележащему RxGo Observable |

**Создание потока:**
```go
stream := reactive.NewStream(ctx, ch, reactive.DefaultStreamConfig())
```

### 2. SimRepository (`internal/reactive/sim_repository.go`)

Реактивный репозиторий для работы с SIM-картами.

**Метод:**
- `GetAllAsStream(ctx)` — получение всех SIM как Observable поток

Используется в `ReactiveSimsListHandler` для потоковой выдачи списка SIM.

### 3. EventBroadcaster (`internal/reactive/event_broadcaster.go`)

Система fan-out событий для SSE-клиентов. Каждый подписчик получает свой канал.

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

**Основные методы:**
- `Subscribe()` / `Unsubscribe()` — управление подписчиками (fan-out)
- `Emit()` — отправка событий во все подписанные каналы
- `GetStats()` — количество подписчиков и событий (атомарные счётчики)

## API Endpoints (4 reactive эндпоинта)

### `GET /api/v1/reactive/events`

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

### `GET /api/v1/reactive/sims`

Список SIM-карт через reactive Observable pipeline.

**Pipeline:** `GetAllAsStream → Collect → JSON`

**Ответ:**
```json
{
  "count": 1086,
  "sims": [
    { "msisdn": "972501234567", "iccid": "8997201...", "status": "Activated" }
  ]
}
```

### `GET /api/v1/reactive/search?q=<query>`

Reactive поиск с прямым DB-запросом. Поддерживает field-specific синтаксис.

| Параметр | Тип | Описание |
|----------|-----|----------|
| `q` | query, required | Строка поиска. `field:value` для конкретного поля или просто `value` для всех полей |

**Поддерживаемые поля:** `msisdn`, `iccid`, `imsi`, `cli`, `status`, `imei`, `ip`, `rateplan`, `label`

**Примеры:**
- `?q=972501` — поиск по всем полям
- `?q=status:Activated` — только по полю Status
- `?q=iccid:8997201` — только по ICCID

**Ответ:**
```json
{
  "query": "972501",
  "count": 15,
  "results": [ ... ]
}
```

> **Примечание:** Debounce реализован на клиенте (React SPA — 350ms, test-reactive.html — 300ms). Серверный эндпоинт выполняет поиск мгновенно.

### `GET /api/v1/reactive/stats`

Агрегированная статистика событий из EventBroadcaster.

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

es.onerror = () => console.log('SSE reconnecting...');
window.addEventListener('beforeunload', () => es.close());
```

### 2. Reactive search с client-side debounce (JavaScript)

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
```

### 3. Field-specific search (JavaScript)

```javascript
// Search only by status
fetch('/api/v1/reactive/search?q=status:Activated', { headers: authHeaders });

// Search only by ICCID
fetch('/api/v1/reactive/search?q=iccid:8997201', { headers: authHeaders });

// Search all fields (default)
fetch('/api/v1/reactive/search?q=972501', { headers: authHeaders });
```

### 4. Go — эмиссия событий из хендлеров

```go
package handlers

import "eyeson-go-server/internal/reactive"

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

### 5. cURL / PowerShell — быстрое тестирование

```bash
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

# Search (all fields)
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:5000/api/v1/reactive/search?q=972501" | jq '.'

# Search (field-specific)
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:5000/api/v1/reactive/search?q=status:Activated" | jq '.'

# Stats
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:5000/api/v1/reactive/stats | jq '.'
```

```powershell
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

| Категория | Тест | Результат |
|-----------|------|-----------|
| Auth | POST /auth/login | PASS |
| Reactive | GET /reactive/sims | PASS (1086 SIM) |
| Reactive | GET /reactive/search (all fields) | PASS |
| Reactive | GET /reactive/search (field-specific) | PASS |
| Reactive | GET /reactive/stats | PASS (null = no events) |
| Reactive | GET /reactive/events (SSE) | PASS |
| Frontend | Reactive search toggle | PASS |
| Frontend | Client-side debounce (350ms) | PASS |

## Тестовая консоль

Интерактивная страница для тестирования reactive endpoints:
- **URL:** `http://localhost:5000/test-reactive.html`
- SSE Stream viewer с метриками подключения
- Live search с field selector и client-side debounce
- Сортируемая таблица результатов с подсветкой совпадений
- Генератор событий (SIM status change, Sync trigger)
- Автоматический тест-suite (7 тестов)
- Примеры кода для JS, Go, cURL

## Преимущества

| Преимущество | Описание |
|-------------|----------|
| Real-time | SSE-события доставляются мгновенно через fan-out broadcaster |
| Scalable SSE | Каждый клиент получает свой канал, не блокируя остальных |
| Atomic Stats | Потокобезопасные счётчики подписчиков и событий |
| Field Search | Поиск по конкретному полю снижает нагрузку на DB |
| Client Debounce | 300–350ms debounce на клиенте, мгновенный ответ сервера |
| Composability | Stream operators (Map, Filter) для трансформации данных |

## Дальнейшее развитие

- [ ] WebSocket поддержка (дополнительно к SSE)
- [ ] Replay механизм для новых подписчиков (event store)
- [ ] Rate limiting для событий
- [ ] Reactive метрики и алерты (Prometheus)
- [ ] Circuit breaker для upstream API
- [ ] Reactive cache invalidation
