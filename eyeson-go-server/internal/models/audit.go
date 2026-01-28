// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package models

import (
	"time"

	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════
// AUDIT CONSTANTS
// ═══════════════════════════════════════════════════════════

// EntityType - тип сущности для аудита
type EntityType string

const (
	EntitySIM     EntityType = "sim"
	EntityUser    EntityType = "user"
	EntityRole    EntityType = "role"
	EntityTask    EntityType = "task"
	EntitySession EntityType = "session"
	EntitySystem  EntityType = "system"
)

// AuditAction - тип действия
type AuditAction string

const (
	ActionCreate        AuditAction = "CREATE"
	ActionUpdate        AuditAction = "UPDATE"
	ActionDelete        AuditAction = "DELETE"
	ActionStatusChange  AuditAction = "STATUS_CHANGE"
	ActionBulkChange    AuditAction = "BULK_CHANGE"
	ActionLogin         AuditAction = "LOGIN"
	ActionLogout        AuditAction = "LOGOUT"
	ActionLoginFailed   AuditAction = "LOGIN_FAILED"
	ActionExport        AuditAction = "EXPORT"
	ActionSync          AuditAction = "SYNC"
	ActionQueueAdd      AuditAction = "QUEUE_ADD"
	ActionQueueRetry    AuditAction = "QUEUE_RETRY"
	ActionQueueComplete AuditAction = "QUEUE_COMPLETE"
	ActionQueueFail     AuditAction = "QUEUE_FAIL"
	ActionQueueCancel   AuditAction = "QUEUE_CANCEL"
)

// AuditSource - источник действия
type AuditSource string

const (
	SourceWeb    AuditSource = "WEB"    // Через веб-интерфейс
	SourceAPI    AuditSource = "API"    // Через REST API
	SourceSync   AuditSource = "SYNC"   // Фоновая синхронизация
	SourceWorker AuditSource = "WORKER" // Job worker
	SourceSystem AuditSource = "SYSTEM" // Системные операции
)

// AuditStatus - статус операции
type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "SUCCESS"
	AuditStatusFailed  AuditStatus = "FAILED"
	AuditStatusPending AuditStatus = "PENDING"
	AuditStatusQueued  AuditStatus = "QUEUED"
)

// ═══════════════════════════════════════════════════════════
// AUDIT LOG MODEL
// ═══════════════════════════════════════════════════════════

// AuditLog - единая модель аудита всех действий в системе
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`

	// ─── КТО ───────────────────────────────────────────────
	UserID    *uint  `gorm:"index" json:"user_id"`           // FK на User (nil для system)
	Username  string `gorm:"index;size:100" json:"username"` // Копия для быстрого доступа
	UserRole  string `gorm:"size:50" json:"user_role"`       // Роль на момент действия
	IPAddress string `gorm:"size:45" json:"ip_address"`      // IPv4 или IPv6
	UserAgent string `gorm:"size:500" json:"user_agent"`     // Browser/Client info

	// ─── ЧТО ───────────────────────────────────────────────
	EntityType EntityType  `gorm:"index;size:30" json:"entity_type"` // sim, user, role...
	EntityID   string      `gorm:"index;size:50" json:"entity_id"`   // MSISDN для SIM, ID для других
	Action     AuditAction `gorm:"index;size:30" json:"action"`      // CREATE, UPDATE, DELETE...

	// ─── ДЕТАЛИ ИЗМЕНЕНИЯ ──────────────────────────────────
	Field     string `gorm:"size:50" json:"field,omitempty"`        // Какое поле изменилось
	OldValue  string `gorm:"type:text" json:"old_value,omitempty"`  // Старое значение
	NewValue  string `gorm:"type:text" json:"new_value,omitempty"`  // Новое значение
	ChangeSet string `gorm:"type:text" json:"change_set,omitempty"` // JSON для bulk операций

	// ─── КОНТЕКСТ ──────────────────────────────────────────
	Source      AuditSource `gorm:"index;size:20" json:"source"`              // WEB, API, SYNC, WORKER
	TaskID      *uint       `gorm:"index" json:"task_id,omitempty"`           // FK на SyncTask
	BatchID     string      `gorm:"index;size:36" json:"batch_id,omitempty"`  // UUID группы операций
	SessionID   string      `gorm:"index;size:64" json:"session_id,omitempty"` // JWT session
	RequestPath string      `gorm:"size:200" json:"request_path,omitempty"`   // /api/v1/sims/bulk

	// ─── РЕЗУЛЬТАТ ─────────────────────────────────────────
	Status       AuditStatus `gorm:"index;size:20" json:"status"`
	ErrorMessage string      `gorm:"type:text" json:"error_message,omitempty"`

	// ─── ПРОВАЙДЕР ─────────────────────────────────────────
	ProviderRequestID  int   `json:"provider_request_id,omitempty"`  // ID от Pelephone
	ProviderResponseMs int64 `json:"provider_response_ms,omitempty"` // Время ответа API
}

// TableName - имя таблицы
func (AuditLog) TableName() string {
	return "audit_logs"
}

// BeforeCreate - хук перед созданием записи
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	return nil
}
