// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// ═══════════════════════════════════════════════════════════
// AUDIT SERVICE
// ═══════════════════════════════════════════════════════════

// AuditService - сервис для логирования действий в системе
type AuditService struct{}

// Audit - глобальный экземпляр сервиса аудита
var Audit = &AuditService{}

// ─── КОНТЕКСТ ПОЛЬЗОВАТЕЛЯ ─────────────────────────────────

// UserContext - информация о текущем пользователе
type UserContext struct {
	UserID    uint
	Username  string
	Role      string
	IPAddress string
	UserAgent string
	SessionID string
}

// GetUserContext извлекает данные пользователя из Fiber context
func (s *AuditService) GetUserContext(c *fiber.Ctx) UserContext {
	ctx := UserContext{
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}

	// Извлекаем из JWT
	if claims, ok := c.Locals("user").(jwt.MapClaims); ok {
		if id, ok := claims["user_id"].(float64); ok {
			ctx.UserID = uint(id)
		}
		if username, ok := claims["username"].(string); ok {
			ctx.Username = username
		}
		if role, ok := claims["role"].(string); ok {
			ctx.Role = role
		}
		if sessionID, ok := claims["session_id"].(string); ok {
			ctx.SessionID = sessionID
		}
	}

	return ctx
}

// ─── LOG BUILDER ───────────────────────────────────────────

// LogBuilder - builder для создания записи аудита
type LogBuilder struct {
	log *models.AuditLog
}

// NewLog создаёт builder для записи аудита с контекстом из запроса
func (s *AuditService) NewLog(c *fiber.Ctx) *LogBuilder {
	user := s.GetUserContext(c)

	userID := user.UserID
	return &LogBuilder{
		log: &models.AuditLog{
			UserID:      &userID,
			Username:    user.Username,
			UserRole:    user.Role,
			IPAddress:   user.IPAddress,
			UserAgent:   user.UserAgent,
			SessionID:   user.SessionID,
			RequestPath: c.Path(),
			Source:      models.SourceWeb,
			Status:      models.AuditStatusSuccess,
			CreatedAt:   time.Now(),
		},
	}
}

// NewWorkerLog создаёт builder для записи от worker'а
func (s *AuditService) NewWorkerLog() *LogBuilder {
	return &LogBuilder{
		log: &models.AuditLog{
			Username:  "worker",
			Source:    models.SourceWorker,
			Status:    models.AuditStatusSuccess,
			CreatedAt: time.Now(),
		},
	}
}

// ─── BUILDER METHODS ───────────────────────────────────────

// Entity устанавливает тип и ID сущности
func (b *LogBuilder) Entity(entityType models.EntityType, entityID string) *LogBuilder {
	b.log.EntityType = entityType
	b.log.EntityID = entityID
	return b
}

// Action устанавливает тип действия
func (b *LogBuilder) Action(action models.AuditAction) *LogBuilder {
	b.log.Action = action
	return b
}

// Change устанавливает детали изменения
func (b *LogBuilder) Change(field, oldValue, newValue string) *LogBuilder {
	b.log.Field = field
	b.log.OldValue = oldValue
	b.log.NewValue = newValue
	return b
}

// ChangeSet устанавливает набор изменений для bulk операций
func (b *LogBuilder) ChangeSet(changes []map[string]string) *LogBuilder {
	data, _ := json.Marshal(changes)
	b.log.ChangeSet = string(data)
	return b
}

// Task связывает запись с задачей в очереди
func (b *LogBuilder) Task(taskID uint) *LogBuilder {
	b.log.TaskID = &taskID
	return b
}

// Batch связывает запись с batch операцией
func (b *LogBuilder) Batch(batchID string) *LogBuilder {
	b.log.BatchID = batchID
	return b
}

// Provider устанавливает данные ответа провайдера
func (b *LogBuilder) Provider(requestID int, responseMs int64) *LogBuilder {
	b.log.ProviderRequestID = requestID
	b.log.ProviderResponseMs = responseMs
	return b
}

// Failed помечает операцию как неудачную
func (b *LogBuilder) Failed(err error) *LogBuilder {
	b.log.Status = models.AuditStatusFailed
	if err != nil {
		b.log.ErrorMessage = err.Error()
	}
	return b
}

// Queued помечает операцию как поставленную в очередь
func (b *LogBuilder) Queued() *LogBuilder {
	b.log.Status = models.AuditStatusQueued
	return b
}

// Pending помечает операцию как ожидающую
func (b *LogBuilder) Pending() *LogBuilder {
	b.log.Status = models.AuditStatusPending
	return b
}

// WithError добавляет сообщение об ошибке
func (b *LogBuilder) WithError(errMsg string) *LogBuilder {
	b.log.ErrorMessage = errMsg
	return b
}

// Save сохраняет запись в БД синхронно
func (b *LogBuilder) Save() error {
	return database.DB.Create(b.log).Error
}

// SaveAsync сохраняет запись асинхронно (fire-and-forget)
func (b *LogBuilder) SaveAsync() {
	go func() {
		if err := b.Save(); err != nil {
			log.Printf("[Audit] Failed to save log: %v", err)
		}
	}()
}

// ─── БЫСТРЫЕ МЕТОДЫ ЛОГИРОВАНИЯ ────────────────────────────


// LogStatusChangeQueued - логирует постановку изменения статуса в очередь
func (s *AuditService) LogStatusChangeQueued(c *fiber.Ctx, cli, msisdn, oldStatus, newStatus string, taskID uint, reason error) {
	builder := s.NewLog(c).
		Entity(models.EntitySIM, msisdn).
		Action(models.ActionQueueAdd).
		Change("status", oldStatus, newStatus).
		Task(taskID).
		Queued()

	if reason != nil {
		builder.WithError(fmt.Sprintf("Queued due to: %s", reason.Error()))
	}

	builder.SaveAsync()
}

// LogBulkStatusChange - логирует массовое изменение статуса
func (s *AuditService) LogBulkStatusChange(c *fiber.Ctx, batchID string, items []map[string]string, newStatus string, providerReqID int, responseMs int64, err error) {
	// Создаём changeset
	changes := make([]map[string]string, 0, len(items))
	for _, item := range items {
		changes = append(changes, map[string]string{
			"cli":        item["cli"],
			"msisdn":     item["msisdn"],
			"old_status": item["old_status"],
			"new_status": newStatus,
		})
	}

	builder := s.NewLog(c).
		Entity(models.EntitySIM, fmt.Sprintf("batch:%d", len(items))).
		Action(models.ActionBulkChange).
		ChangeSet(changes).
		Batch(batchID).
		Provider(providerReqID, responseMs)

	if err != nil {
		builder.Failed(err)
	}

	builder.SaveAsync()
}

// LogLogin - логирует вход пользователя
func (s *AuditService) LogLogin(c *fiber.Ctx, userID uint, username, role string, success bool, errMsg string) {
	action := models.ActionLogin
	if !success {
		action = models.ActionLoginFailed
	}

	builder := &LogBuilder{
		log: &models.AuditLog{
			UserID:      &userID,
			Username:    username,
			UserRole:    role,
			IPAddress:   c.IP(),
			UserAgent:   c.Get("User-Agent"),
			RequestPath: c.Path(),
			EntityType:  models.EntitySession,
			EntityID:    username,
			Action:      action,
			Source:      models.SourceWeb,
			CreatedAt:   time.Now(),
		},
	}

	if success {
		builder.log.Status = models.AuditStatusSuccess
	} else {
		builder.log.Status = models.AuditStatusFailed
		builder.log.ErrorMessage = errMsg
	}

	builder.SaveAsync()
}

// LogQueueCompleted - логирует успешное выполнение задачи из очереди
func (s *AuditService) LogQueueCompleted(taskID uint, msisdn, result string, responseMs int64) {
	s.NewWorkerLog().
		Entity(models.EntitySIM, msisdn).
		Action(models.ActionQueueComplete).
		Task(taskID).
		Provider(0, responseMs).
		SaveAsync()
}

// LogQueueFailed - логирует неудачу задачи из очереди
func (s *AuditService) LogQueueFailed(taskID uint, msisdn, errMsg string, durationMs int64) {
	s.NewWorkerLog().
		Entity(models.EntitySIM, msisdn).
		Action(models.ActionQueueFail).
		Task(taskID).
		Failed(errors.New(errMsg)).
		SaveAsync()
}

// LogQueueRetry - логирует повторную попытку выполнения задачи
func (s *AuditService) LogQueueRetry(taskID uint, msisdn string, attempt, maxAttempts int, errMsg string) {
	s.NewWorkerLog().
		Entity(models.EntitySIM, msisdn).
		Action(models.ActionQueueRetry).
		Task(taskID).
		WithError(fmt.Sprintf("Attempt %d/%d: %s", attempt, maxAttempts, errMsg)).
		Pending().
		SaveAsync()
}

// LogQueueCancel - логирует отмену задачи
func (s *AuditService) LogQueueCancel(c *fiber.Ctx, task *models.SyncTaskExtended) {
	s.NewLog(c).
		Entity(models.EntitySIM, task.TargetMSISDN).
		Action(models.ActionQueueCancel).
		Task(task.ID).
		SaveAsync()
}

// LogUserCreate - логирует создание пользователя
func (s *AuditService) LogUserCreate(c *fiber.Ctx, newUserID uint, newUsername string) {
	s.NewLog(c).
		Entity(models.EntityUser, fmt.Sprintf("%d", newUserID)).
		Action(models.ActionCreate).
		Change("username", "", newUsername).
		SaveAsync()
}

// LogUserUpdate - логирует обновление пользователя
func (s *AuditService) LogUserUpdate(c *fiber.Ctx, userID uint, field, oldValue, newValue string) {
	s.NewLog(c).
		Entity(models.EntityUser, fmt.Sprintf("%d", userID)).
		Action(models.ActionUpdate).
		Change(field, oldValue, newValue).
		SaveAsync()
}

// LogUserDelete - логирует удаление пользователя
func (s *AuditService) LogUserDelete(c *fiber.Ctx, userID uint, username string) {
	s.NewLog(c).
		Entity(models.EntityUser, fmt.Sprintf("%d", userID)).
		Action(models.ActionDelete).
		Change("username", username, "").
		SaveAsync()
}

// LogExport - логирует экспорт данных
func (s *AuditService) LogExport(c *fiber.Ctx, exportType string, recordCount int) {
	s.NewLog(c).
		Entity(models.EntitySystem, exportType).
		Action(models.ActionExport).
		Change("count", "", fmt.Sprintf("%d", recordCount)).
		SaveAsync()
}
