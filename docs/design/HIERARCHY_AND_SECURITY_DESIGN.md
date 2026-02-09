# Дизайн системы иерархии, безопасности и биллинга (CMSV Integration)

Этот документ описывает архитектуру для безопасного маппинга иерархии CMSV (Дилер -> Компания -> Пользователь) в личный кабинет, обеспечивая изоляцию данных и защиту от несанкционированных оплат.

---

## 1. Концепция "Зеркальной Иерархии" (Mirror Hierarchy)

Мы не просто храним плоский список пользователей, мы реплицируем дерево владения из CMSV.

*   **Eyeson Cabinet** — это "Верхний уровень".
*   **CMSV** — это "Поставщик ресурсов" (устройства).
*   **Billing** — это "Правила доступа".

### Уровни доступа:
1.  **Super Admin (Eyeson):** Видит всё.
2.  **Dealer (Дилер):** Видит свои Компании и их устройства. Может платить за всех "вниз" по цепочке.
3.  **Company Admin (Владелец парка):** Видит только свои машины. Платит за них.
4.  **End User (Водитель/Менеджер):** Видит только машины, назначенные лично ему. Не платит (или платит только за свои).

---

## 2. Модель Данных (Database Extensions)

Добавляем иерархичность в `internal/models/hierarchy.go`.

### A. Organization (Узел иерархии)
Аналог "Группы" или "Компании" в CMSV.

```go
type Organization struct {
    ID             uint   `gorm:"primaryKey"`
    Name           string // Напр. "Taxi Service Ltd."
    Type           string // "Dealer", "Company", "Department"
    ParentID       *uint  // Ссылка на родителя (Дилера)
    
    // Связь с CMSV
    CmsvGroupID    string `gorm:"uniqueIndex"` // ID группы в базе CMSV
    CmsvLogin      string // Логин админа этой группы (для синхронизации)
    
    // Финансы
    BalanceILS     float64 // Общий баланс компании
    IsPrepaid      bool    // true = предоплата, false = постоплата (счет в конце месяца)
}
```

### B. Device (Устройство в контексте безопасности)
Связывает железо с иерархией.

```go
type Device struct {
    ID             uint   `gorm:"primaryKey"`
    SimCardID      uint   // Ссылка на SIM (Pelephone)
    OrganizationID uint   // Чье это устройство (Компания)
    UserID         *uint  // Личная привязка (если водитель платит сам)
    
    CmsvDeviceID   string `gorm:"uniqueIndex"` // "8800..."
    Status         string // "Active", "Blocked_By_Billing"
}
```

---

## 3. Логика Идентификации (Linkage Flow)

Как мы гарантируем, что Клиент А не привяжет устройства Клиента Б?

**Процесс "Claim Ownership" (Заявка на владение):**

1.  **Вход в Кабинет:** Клиент регистрируется в Eyeson (Email/Pass).
2.  **Привязка CMSV:**
    *   Клиент вводит `CMSV Login` и `CMSV Password`.
    *   **Бэкенд:** Идет в CMSV API -> `Login()`.
    *   *Если успех:* Получает `UserToken` и `RootGroupID` (корневую группу этого юзера).
3.  **Создание Связи:**
    *   Бэкенд создает запись `Organization` (или находит существующую).
    *   Записывает: "Этот Eyeson User является Админом этой CMSV Organization".
4.  **Синхронизация:**
    *   Скачивает все дочерние устройства рекурсивно.
    *   Помечает их `OrganizationID`.

**Защита:** Так как мы требуем пароль от CMSV, пользователь не может привязать чужую компанию, пароля которой он не знает.

---

## 4. Политика Оплаты (Billing Policy)

Реализовать middleware или сервис проверку `CanPayFor`.

**Правило:** Платить за устройство может только тот, кто находится в одной ветке иерархии *выше* устройства.

```go
func (s *BillingService) CanPayFor(payerUser models.User, deviceID uint) bool {
    var device models.Device
    db.First(&device, deviceID)

    // 1. Прямое владение (Водитель платит за свою машину)
    if device.UserID != nil && *device.UserID == payerUser.ID {
        return true
    }

    // 2. Корпоративное владение (Админ платит за машины компании)
    // Проверяем, принадлежит ли юзер к организации-владельцу устройства
    if s.IsUserAdminOfOrg(payerUser.ID, device.OrganizationID) {
        return true
    }

    // 3. Дилерское владение (Дилер платит за машины суб-компании)
    if s.IsUserAdminOfParentOrg(payerUser.ID, device.OrganizationID) {
        return true
    }

    return false
}
```

---

## 5. Сценарий: "Защита от Дурака" (Double Payment Prevention)

**Проблема:** И Дилер, и Компания, и Водитель видят устройство. Кто платит?

**Решение:** Приоритет плательщика (Payer Priority).
1.  В настройках `Organization` ставится флаг `BillingMode`:
    *   **Centralized:** Платит только Голова (Дилер). Кнопки оплаты у дочерних юзеров скрыты.
    *   **Distributed:** Каждая дочка платит сама. Дилер видит статус "Не оплачено", но платить не обязан.

**Реализация:**
При отображении кнопки "Купить подписку" на фронтенде:
```javascript
if (device.BillingMode === 'Centralized' && currentUser.Role !== 'Dealer') {
    showButton = false;
    showText = "Оплата производится администратором";
}
```

---

## 6. Итоговый алгоритм безопасности

1.  **Auth:** Вход в Eyeson -> JWT Token.
2.  **Scope:** В токене зашит `OrgID`.
3.  **Query:** При запросе `GET /devices` бэкенд делает фильтр `WHERE organization_id IN (child_orgs_of(user.OrgID))`.
    *   *Это гарантирует, что чужие устройства физически не видны.*
4.  **Action:** При попытке `POST /pay` выполняется проверка `CanPayFor`.
5.  **Execution:** Деньги списываются с карты User-а, подписка вешается на Device.
