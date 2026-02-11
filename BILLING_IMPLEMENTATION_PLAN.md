# План построения Биллингового кабинета на базе Eyeson-Go (CMSV + EasyCard + Pelephone)

## 1. Текущее состояние (Audit)
Проект `eyeson-go-server` уже реализует **технический уровень** управления SIM-картами:
*   **Pelephone Integration:** Полностью рабочий клиент (`internal/eyesont/client.go`), умеющий получать список SIM (`GetSims`) и менять их статус (`BulkUpdate`).
*   **Database:** Есть таблица `SimCard` с техническими данными (ICCID, Status, UsageMB), но **нет** финансовых данных (Цена, Тариф, Баланс).
*   **API:** Реализованы эндпоинты для управления статусами (`/sims/status`), но **нет** платежных шлюзов.

## 2. Архитектура решения (Target Architecture)
Кабинет пользователя будет состоять из трех слоев:
1.  **Billing Engine (Ядро):** Отвечает за тарифы, подписки и сроки действия услуг.
2.  **Payment Gateway (EasyCard):** Обрабатывает транзакции (списание средств, токенизация карт).
3.  **Connectivity Manager (Pelephone):** Исполнительный механизм (блокирует/разблокирует SIM в зависимости от оплаты).

---

## 3. План реализации (Implementation Roadmap)

### Этап 1: Расширение Модели Данных (Data Layer)
Необходимо добавить финансовые сущности в `internal/models/db.go`:

1.  **Таблица `TariffPlan` (Тарифы):**
    *   Связывает технический `RatePlan` (из Pelephone) с деньгами.
    *   Поля: `ID`, `PelephonePlanName` (строка, для связи), `PriceILS` (цена), `BillingPeriod` (месяц/год).

2.  **Таблица `Customer` (Клиент):**
    *   Владелец SIM-карт и плательщик.
    *   Поля: `ID`, `EasyCardToken` (токен карты для автоплатежей), `Email`, `Name`, `Phone`.

3.  **Доработка таблицы `SimCard`:**
    *   `CustomerID` (Foreign Key -> Customer).
    *   `PaidUntil` (Date: дата окончания оплаченного периода).
    *   `IsAutoRenew` (Bool: автопродление).
    *   `LastPaymentStatus` (Enum: Paid, Failed, Pending).

4.  **Таблица `Transaction` (История платежей):**
    *   Поля: `ID`, `CustomerID`, `SimID`, `Amount`, `Status`, `EasyCardRef`, `CreatedAt`.

### Этап 2: Интеграция EasyCard (Service Layer)
Создать новый пакет `internal/services/payment`:

1.  **EasyCard Client:**
    *   Реализовать методы на основе `swagger.yaml` (который мы подготовили ранее).
    *   `CreatePaymentIntent(amount, currency, redirectUrl)` — для первичной привязки карты.
    *   `CreateTransaction(token, amount)` — для рекуррентных списаний.
    *   `GetTransactionStatus(id)` — проверка статуса.

2.  **Webhook Handler:**
    *   В `internal/handlers` добавить эндпоинт `POST /webhooks/easycard`.
    *   При получении уведомления `TransactionSuccess` -> Найти транзакцию -> Обновить `SimCard.PaidUntil`.

### Этап 3: Биллинговый Движок (Business Logic)
Создать `internal/jobs/billing_worker.go` (фоновый процесс):

1.  **Ежедневная задача (Cron):**
    *   Выбрать все SIM, где `PaidUntil < Tomorrow` и `IsAutoRenew = true`.
    *   Для каждой SIM найти `Customer` и `TariffPlan`.
    *   **Действие:** Вызвать `PaymentService.ChargeToken(Customer.EasyCardToken, TariffPlan.PriceILS)`.

2.  **Сценарий "Успех":**
    *   Продлить `PaidUntil` на +1 месяц.
    *   Создать запись в `Transaction` со статусом `Success`.
    *   Если SIM была `Suspended` (заблокирована), вызвать `EyesonT.BulkUpdate(..., "Active")` для разблокировки.

3.  **Сценарий "Провал" (нет денег / карта истекла):**
    *   Пометить `LastPaymentStatus = Failed`.
    *   Отправить Email/SMS клиенту.
    *   Если просрочка > 3 дней (Grace Period), вызвать `EyesonT.BulkUpdate(..., "Suspend")` для блокировки SIM.

### Этап 4: API Кабинета (Handlers)
Добавить новые эндпоинты для фронтенда в `internal/handlers`:

*   `GET /billing/plans` — список доступных тарифов.
*   `POST /billing/subscribe` — привязать SIM к тарифу.
*   `GET /billing/invoices` — история списаний.
*   `POST /billing/pay` — ручная оплата долга.

---

## 4. Рекомендация по стеку
Текущий проект на Go (`eyeson-go-server`) имеет правильную структуру (`handlers`, `services`, `models`).
**Не нужно** создавать отдельный микросервис. Интегрируйте логику биллинга прямо в `internal/services`, чтобы использовать уже существующий клиент Pelephone (`eyesont`) без лишних усложнений.
