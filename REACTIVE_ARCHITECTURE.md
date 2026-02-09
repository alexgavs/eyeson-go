# Reactive Programming Architecture

## Обзор

Проект EyesOn переведен на реактивную архитектуру с использованием RxGo для обработки потоков данных в реальном времени.

## Ключевые компоненты

### 1. Reactive Stream (`internal/reactive/stream.go`)
Базовый класс для работы с реактивными потоками данных.

**Основные операторы:**
- `Map` - трансформация данных
- `Filter` - фильтрация элементов
- `FlatMap` - преобразование в подпотоки
- `Buffer` - группировка элементов
- `Debounce` - задержка между элементами
- `Distinct` - удаление дубликатов
- `Retry` - повторные попытки при ошибках
- `CatchError` - обработка ошибок

### 2. SimRepository (`internal/reactive/sim_repository.go`)
Реактивный репозиторий для работы с SIM-картами.

**Возможности:**
- `GetAllAsStream()` - получение всех SIM как поток
- `WatchChanges()` - мониторинг изменений в реальном времени
- `FindByStatusStream()` - фильтрация по статусу
- `BatchUpdate()` - пакетное обновление с буферизацией
- `SearchStream()` - поиск с debouncing (300ms)

### 3. SyncService (`internal/reactive/sync_service.go`)
Реактивная синхронизация с upstream API.

**Возможности:**
- `ProcessTaskStream()` - обработка задач с retry и timeout
- `SyncAllSims()` - полная синхронизация
- `PeriodicSync()` - периодическая синхронизация
- `MonitorChanges()` - мониторинг и авто-синхронизация изменений

### 4. EventBroadcaster (`internal/reactive/event_broadcaster.go`)
Система реактивных событий для SSE.

**Типы событий:**
- `SIM_CREATED`, `SIM_UPDATED`, `SIM_DELETED`
- `SYNC_STARTED`, `SYNC_COMPLETED`, `SYNC_FAILED`
- `TASK_QUEUED`, `TASK_PROCESSING`, `TASK_COMPLETED`, `TASK_FAILED`

**Возможности:**
- `Emit()` - отправка событий
- `FilterByType()` - фильтрация по типу событий
- `FilterByUser()` - фильтрация по пользователю
- `ToSSE()` - конвертация в SSE формат
- `AggregateStats()` - агрегация статистики

## API Endpoints

### Реактивные эндпоинты

```
GET /api/v1/reactive/events
```
Server-Sent Events поток событий в реальном времени.
- Query params: `user_id`, `types`

```
GET /api/v1/reactive/sims
```
Список SIM-карт через reactive stream.

```
GET /api/v1/reactive/search?q=<query>
```
Реактивный поиск с debouncing.

```
GET /api/v1/reactive/stats
```
Агрегированная статистика событий.

## Примеры использования

### 1. Подписка на события
```javascript
const eventSource = new EventSource('/api/v1/reactive/events?user_id=admin');
eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data);
};
```

### 2. Реактивный поиск
```javascript
const searchInput = document.getElementById('search');
let debounceTimer;

searchInput.addEventListener('input', (e) => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(async () => {
    const response = await fetch(`/api/v1/reactive/search?q=${e.target.value}`);
    const data = await response.json();
    updateResults(data.results);
  }, 300);
});
```

### 3. Мониторинг статистики
```javascript
setInterval(async () => {
  const response = await fetch('/api/v1/reactive/stats');
  const stats = await response.json();
  updateDashboard(stats);
}, 5000);
```

## Преимущества реактивной архитектуры

✅ **Real-time обновления** - события доставляются мгновенно через SSE  
✅ **Backpressure control** - контроль потока данных через буферизацию  
✅ **Error handling** - автоматические retry и fallback  
✅ **Debouncing** - снижение нагрузки на поиск и API  
✅ **Batch processing** - эффективная обработка пакетов данных  
✅ **Scalability** - параллельная обработка через stream operators  

## Дальнейшее развитие

- [ ] Добавить WebSocket поддержку
- [ ] Реализовать replay механизм для новых подписчиков
- [ ] Добавить rate limiting для событий
- [ ] Реактивные метрики и алерты
- [ ] Circuit breaker для upstream API
- [ ] Кеширование с reactive invalidation
