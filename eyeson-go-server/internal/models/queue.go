// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════
// QUEUE CONSTANTS
// ═══════════════════════════════════════════════════════════

// TaskType - тип задачи в очереди
type TaskType string

const (
	TaskTypeStatusChange TaskType = "STATUS_CHANGE"
	TaskTypeLabelUpdate  TaskType = "LABEL_UPDATE"
	TaskTypeBulkChange   TaskType = "BULK_CHANGE"
	TaskTypeSync         TaskType = "SYNC"
)

// TaskStatus - статус задачи
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusProcessing TaskStatus = "PROCESSING"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
	TaskStatusFailed     TaskStatus = "FAILED"
	TaskStatusCancelled  TaskStatus = "CANCELLED"
)

// TaskPriority - приоритет задачи (1 = высший, 10 = низший)
type TaskPriority int

const (
	PriorityUrgent     TaskPriority = 1  // Критические операции
	PriorityHigh       TaskPriority = 3  // Пользовательские действия
	PriorityNormal     TaskPriority = 5  // Обычные операции
	PriorityLow        TaskPriority = 7  // Фоновые задачи
	PriorityBackground TaskPriority = 10 // Синхронизация
)

// ═══════════════════════════════════════════════════════════
// EXTENDED SYNC TASK MODEL
// ═══════════════════════════════════════════════════════════

// SyncTaskExtended - расширенная модель задачи в очереди
// Заменяет/дополняет базовую SyncTask
type SyncTaskExtended struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// ─── ТИП И ПРИОРИТЕТ ───────────────────────────────────
	Type     TaskType     `gorm:"index;size:30" json:"type"`
	Priority TaskPriority `gorm:"index;default:5" json:"priority"`
	Status   TaskStatus   `gorm:"index;size:20;default:'PENDING'" json:"status"`

	// ─── ДАННЫЕ ЗАДАЧИ ─────────────────────────────────────
	TargetMSISDN string `gorm:"index;size:20" json:"target_msisdn"` // Основной MSISDN
	TargetCLI    string `gorm:"index;size:20" json:"target_cli"`    // CLI карты
	Payload      string `gorm:"type:text" json:"payload"`           // JSON с деталями

	// Для STATUS_CHANGE
	OldStatus string `gorm:"size:50" json:"old_status,omitempty"`
	NewStatus string `gorm:"size:50" json:"new_status,omitempty"`

	// Для LABEL_UPDATE
	LabelField string `gorm:"size:20" json:"label_field,omitempty"`  // label_1, label_2, label_3
	LabelValue string `gorm:"size:200" json:"label_value,omitempty"` // Новое значение метки

	// ─── BATCH (ГРУППОВЫЕ ОПЕРАЦИИ) ────────────────────────
	BatchID    string `gorm:"index;size:36" json:"batch_id,omitempty"` // UUID группы
	BatchTotal int    `json:"batch_total,omitempty"`                   // Всего в группе
	BatchIndex int    `json:"batch_index,omitempty"`                   // Позиция в группе

	// ─── КТО СОЗДАЛ ────────────────────────────────────────
	UserID    *uint  `gorm:"index" json:"user_id"`                     // FK на User
	Username  string `gorm:"size:100" json:"username"`                 // Копия для отображения
	IPAddress string `gorm:"size:45" json:"ip_address"`                // IP клиента
	RequestID string `gorm:"index;size:36" json:"request_id,omitempty"` // Correlation ID для frontend

	// ─── RETRY ЛОГИКА ──────────────────────────────────────
	Attempt     int        `gorm:"default:0" json:"attempt"`
	MaxAttempts int        `gorm:"default:5" json:"max_attempts"`
	NextRunAt   *time.Time `gorm:"index" json:"next_run_at"`
	LastError   string     `gorm:"type:text" json:"last_error,omitempty"`

	// ─── TIMING ────────────────────────────────────────────
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DurationMs  int64      `json:"duration_ms,omitempty"` // Время выполнения

	// ─── РЕЗУЛЬТАТ ─────────────────────────────────────────
	Result            string `gorm:"type:text" json:"result,omitempty"` // Результат или ошибка
	ProviderRequestID int    `json:"provider_request_id,omitempty"`     // ID от Pelephone
}

// TableName - используем ту же таблицу sync_tasks
func (SyncTaskExtended) TableName() string {
	return "sync_tasks"
}

// BeforeCreate - хук перед созданием
func (t *SyncTaskExtended) BeforeCreate(tx *gorm.DB) error {
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	if t.NextRunAt == nil {
		now := time.Now()
		t.NextRunAt = &now
	}
	if t.RequestID == "" {
		t.RequestID = uuid.New().String()
	}
	return nil
}

// CalculateNextRetry - рассчитывает время следующей попытки
// Использует экспоненциальный backoff: 30s, 60s, 120s, 240s, 480s
func (t *SyncTaskExtended) CalculateNextRetry() time.Time {
	baseDelay := 30 * time.Second
	multiplier := 1 << t.Attempt // 2^attempt
	delay := baseDelay * time.Duration(multiplier)

	// Максимум 10 минут между попытками
	maxDelay := 10 * time.Minute
	if delay > maxDelay {
		delay = maxDelay
	}

	return time.Now().Add(delay)
}

// CanRetry - проверяет можно ли повторить попытку
func (t *SyncTaskExtended) CanRetry() bool {
	return t.Attempt < t.MaxAttempts && t.Status != TaskStatusCancelled
}

// IsPending - проверяет находится ли задача в ожидании
func (t *SyncTaskExtended) IsPending() bool {
	return t.Status == TaskStatusPending
}

// IsCompleted - проверяет завершена ли задача
func (t *SyncTaskExtended) IsCompleted() bool {
	return t.Status == TaskStatusCompleted
}

// IsFailed - проверяет провалена ли задача
func (t *SyncTaskExtended) IsFailed() bool {
	return t.Status == TaskStatusFailed
}

// IsActive - проверяет активна ли задача (pending или processing)
func (t *SyncTaskExtended) IsActive() bool {
	return t.Status == TaskStatusPending || t.Status == TaskStatusProcessing
}

// ═══════════════════════════════════════════════════════════
// QUEUE STATISTICS
// ═══════════════════════════════════════════════════════════

// QueueStats - статистика очереди
type QueueStats struct {
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
	Cancelled  int64 `json:"cancelled"`
	TodayTotal int64 `json:"today_total"`
}

// BatchProgress - прогресс выполнения batch операции
type BatchProgress struct {
	BatchID   string  `json:"batch_id"`
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Failed    int     `json:"failed"`
	Pending   int     `json:"pending"`
	Progress  float64 `json:"progress"` // Процент выполнения
}
