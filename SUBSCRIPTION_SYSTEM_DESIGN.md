# Архитектура системы подписок и оптимизации тарифов Eyeson

Этот документ описывает реализацию системы тарифных планов, которая разделяет стоимость продажи (для клиента) и стоимость закупки (у провайдера Pelephone), позволяя максимизировать прибыль за счет автоматического выбора оптимального технического пакета.

---

## 1. Модель данных (Database Schema)

Необходимо добавить следующие структуры в `internal/models/billing.go` (новый файл):

### A. ProductPlan (Продукт для клиента)
Определяет тарифную сетку, видимую в личном кабинете.

```go
package models

import "time"

type ProductPlan struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	Name        string  `json:"name"`          // Напр: "Optimum 10GB", "Camera Only"
	PriceILS    float64 `json:"price_ils"`     // Напр: 69.00
	DataLimitMB uint64  `json:"data_limit_mb"` // Напр: 10240 (10 ГБ)
	IsUnlimited bool    `json:"is_unlimited"`  // true для тарифа за 200 шек
	
	// Права доступа в CMSV
	CanViewVideo   bool `json:"can_view_video"`
	CanViewGPS     bool `json:"can_view_gps"`
	CanViewHistory bool `json:"can_view_history"`
}
```

### B. SupplierPlan (Тариф провайдера)
Технические пакеты, доступные в договоре с Pelephone.

```go
type SupplierPlan struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	ProviderCode string  `json:"provider_code"` // Код тарифа в API Pelephone (напр. "P_15GB_CORP")
	Name         string  `json:"name"`          // Внутреннее название
	CostILS      float64 `json:"cost_ils"`      // Себестоимость (25.00)
	MaxDataMB    uint64  `json:"max_data_mb"`   // Лимит трафика от оператора (15360)
}
```

### C. Subscription (Активная подписка)
Связывает клиента, сим-карту и оба плана.

```go
type Subscription struct {
	ID             uint      `gorm:"primaryKey"`
	CustomerID     uint      `json:"customer_id"`
	SimCardID      uint      `json:"sim_card_id"`
	
	ProductPlanID  uint      // ID тарифа, за который платит клиент
	SupplierPlanID uint      // ID технического тарифа, который мы купили
	
	CurrentUsageMB uint64    // Потребленный трафик за текущий период
	StartDate      time.Time `json:"start_date"`
	NextResetDate  time.Time `json:"next_reset_date"` // Дата сброса счетчика (обычно 1 число месяца)
	Status         string    // "Active", "Throttled", "Blocked", "Expired"
}
```

---

## 2. Логика "Оптимизатора" (Cost Optimizer)

Алгоритм выбора наилучшего технического плана при покупке клиентом услуги.
Реализовать в `internal/services/billing/optimizer.go`.

### Алгоритм `FindBestSupplierPlan`:
1.  **Вход:** Выбранный клиентом `ProductPlan` (например, 10 ГБ).
2.  **Поиск:** Выбрать из `SupplierPlan` все записи, где `MaxDataMB >= ProductPlan.DataLimitMB`.
3.  **Сортировка:** Отсортировать найденные по `CostILS` (по возрастанию).
4.  **Выбор:** Взять первый (самый дешевый).
5.  **Пример:**
    *   Клиент хочет 10 ГБ.
    *   У Pelephone есть: "5 ГБ" (20₪), "15 ГБ" (25₪), "50 ГБ" (40₪).
    *   Фильтр `>= 10 ГБ` оставляет: "15 ГБ" и "50 ГБ".
    *   Выбор: "15 ГБ" за 25₪.
    *   Мы продаем клиенту "10 ГБ" за 69₪. Технически у него есть запас, но мы ограничиваем его программно.

---

## 3. Мониторинг трафика (Usage Monitor)

Фоновая задача для контроля лимитов. Реализовать в `internal/jobs/usage_monitor.go`.

**Логика Job (запуск каждый час):**
1.  Получить список активных `Subscription`.
2.  Для каждой подписки:
    *   Получить актуальный трафик с SIM-карты через `eyesont.GetSims()`.
    *   Обновить `Subscription.CurrentUsageMB`.
3.  **Проверка условий:**
    *   Если `CurrentUsageMB >= ProductPlan.DataLimitMB` И `!IsUnlimited`:
        *   **Действие 1:** Блокировка видео в CMSV (API вызов `DisableVideo`).
        *   **Действие 2:** Отправка уведомления (Email/Push): "Ваш пакет 10 ГБ закончился".
        *   **Действие 3 (Опционально):** Если перерасход критический (>110% от SupplierPlan), приостановить SIM в Pelephone (`Suspend`), чтобы избежать штрафов от оператора.

---

## 4. API Эндпоинты (Handlers)

Добавить в `internal/routes/routes.go`:

### Публичные (для выбора тарифа)
*   `GET /api/plans` — Возвращает список `ProductPlan` (39₪, 49₪, 69₪...).

### Приватные (Кабинет пользователя)
*   `GET /api/my-subscription` — Текущий тариф, дата окончания, статус.
*   `GET /api/my-usage` — Статистика:
    ```json
    {
        "plan_name": "Pro 10GB",
        "used_mb": 8500,
        "total_mb": 10240,
        "percentage": 83
    }
    ```
*   `POST /api/subscribe` — Смена тарифа (вызывает EasyCard для оплаты + Optimizer для смены тех. плана).

---

## 5. Сценарий работы (User Flow)

1.  **Выбор:** Пользователь в кабинете видит "Пакет 10 ГБ за 69₪". Нажимает "Купить".
2.  **Оплата:** Переход на EasyCard, оплата картой.
3.  **Оптимизация:** Бэкенд видит успешную оплату. Ищет в базе Pelephone самый дешевый тариф, покрывающий 10 ГБ (например, "Корпоративный 15 ГБ").
4.  **Провижининг:** Бэкенд шлет команду в Pelephone API: "Перевести SIM ICCID=... на тариф 'Корпоративный 15 ГБ'".
5.  **Активация:** Включает доступ к камерам в CMSV.
6.  **Контроль:** Когда пользователь скачает 10 ГБ, мы отключим ему камеры, хотя технически на симке еще осталось 5 ГБ. Это наш "буфер безопасности" от переплат провайдеру.
