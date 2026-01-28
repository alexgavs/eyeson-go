// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package services

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════════
// QUEUE SERVICE
// ═══════════════════════════════════════════════════════════

// QueueService - сервис для управления очередью задач
type QueueService struct {
	mu sync.RWMutex
}

// Queue - глобальный экземпляр сервиса очереди
var Queue = &QueueService{}

// ─── СОЗДАНИЕ ЗАДАЧ ────────────────────────────────────────

// CreateTaskRequest - запрос на создание задачи
type CreateTaskRequest struct {
	Type       models.TaskType
	Priority   models.TaskPriority
	MSISDN     string
	CLI        string
	OldStatus  string
	NewStatus  string
	LabelField string
	LabelValue string
	UserID     uint
	Username   string
	IPAddress  string
	RequestID  string // Correlation ID от frontend
}

// CreateTask создаёт одну задачу в очереди
func (s *QueueService) CreateTask(req CreateTaskRequest) (*models.SyncTaskExtended, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Генерируем request_id если не передан
	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}

	// Устанавливаем приоритет по умолчанию
	if req.Priority == 0 {
		req.Priority = models.PriorityHigh
	}

	task := &models.SyncTaskExtended{
		Type:         req.Type,
		Priority:     req.Priority,
		Status:       models.TaskStatusPending,
		TargetMSISDN: req.MSISDN,
		TargetCLI:    req.CLI,
		OldStatus:    req.OldStatus,
		NewStatus:    req.NewStatus,
		LabelField:   req.LabelField,
		LabelValue:   req.LabelValue,
		UserID:       &req.UserID,
		Username:     req.Username,
		IPAddress:    req.IPAddress,
		RequestID:    req.RequestID,
		MaxAttempts:  5,
		Attempt:      0,
	}

	// Payload для дополнительных данных
	payload := map[string]interface{}{
		"type":       req.Type,
		"msisdn":     req.MSISDN,
		"cli":        req.CLI,
		"old_status": req.OldStatus,
		"new_status": req.NewStatus,
	}
	payloadJSON, _ := json.Marshal(payload)
	task.Payload = string(payloadJSON)

	// Время первого запуска - сразу
	now := time.Now()
	task.NextRunAt = &now

	if err := database.DB.Create(task).Error; err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	log.Printf("[Queue] Created task #%d: %s %s → %s for %s (request_id: %s)",
		task.ID, task.Type, task.OldStatus, task.NewStatus, task.TargetMSISDN, task.RequestID)

	return task, nil
}

// CreateBatch создаёт группу задач (batch операция)
func (s *QueueService) CreateBatch(items []CreateTaskRequest) (string, []uint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	batchID := uuid.New().String()
	taskIDs := make([]uint, 0, len(items))

	tx := database.DB.Begin()

	for i, req := range items {
		// Генерируем request_id если не передан
		if req.RequestID == "" {
			req.RequestID = uuid.New().String()
		}

		// Устанавливаем приоритет по умолчанию
		if req.Priority == 0 {
			req.Priority = models.PriorityHigh
		}

		task := &models.SyncTaskExtended{
			Type:         req.Type,
			Priority:     req.Priority,
			Status:       models.TaskStatusPending,
			TargetMSISDN: req.MSISDN,
			TargetCLI:    req.CLI,
			OldStatus:    req.OldStatus,
			NewStatus:    req.NewStatus,
			UserID:       &req.UserID,
			Username:     req.Username,
			IPAddress:    req.IPAddress,
			RequestID:    req.RequestID,
			BatchID:      batchID,
			BatchTotal:   len(items),
			BatchIndex:   i + 1,
			MaxAttempts:  5,
			Attempt:      0,
		}

		now := time.Now()
		task.NextRunAt = &now

		if err := tx.Create(task).Error; err != nil {
			tx.Rollback()
			return "", nil, fmt.Errorf("failed to create task %d: %w", i, err)
		}

		taskIDs = append(taskIDs, task.ID)
	}

	if err := tx.Commit().Error; err != nil {
		return "", nil, fmt.Errorf("failed to commit batch: %w", err)
	}

	log.Printf("[Queue] Created batch %s with %d tasks", batchID, len(items))

	return batchID, taskIDs, nil
}

// ─── ПОЛУЧЕНИЕ ЗАДАЧ ───────────────────────────────────────

// GetPendingTasks возвращает задачи готовые к выполнению
func (s *QueueService) GetPendingTasks(limit int) ([]models.SyncTaskExtended, error) {
	var tasks []models.SyncTaskExtended
	now := time.Now()

	err := database.DB.
		Where("status = ? AND (next_run_at IS NULL OR next_run_at <= ?)",
			models.TaskStatusPending, now).
		Order("priority ASC, created_at ASC"). // Сначала высокий приоритет (меньше число)
		Limit(limit).
		Find(&tasks).Error

	return tasks, err
}

// GetTaskByID возвращает задачу по ID
func (s *QueueService) GetTaskByID(id uint) (*models.SyncTaskExtended, error) {
	var task models.SyncTaskExtended
	err := database.DB.Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTaskByRequestID возвращает задачу по correlation ID
func (s *QueueService) GetTaskByRequestID(requestID string) (*models.SyncTaskExtended, error) {
	var task models.SyncTaskExtended
	err := database.DB.Where("request_id = ?", requestID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetUserTasks возвращает задачи пользователя
func (s *QueueService) GetUserTasks(userID uint, statuses []models.TaskStatus) ([]models.SyncTaskExtended, error) {
	var tasks []models.SyncTaskExtended

	query := database.DB.Where("user_id = ?", userID)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	err := query.Order("created_at DESC").Limit(50).Find(&tasks).Error
	return tasks, err
}

// GetBatchTasks возвращает все задачи в batch
func (s *QueueService) GetBatchTasks(batchID string) ([]models.SyncTaskExtended, error) {
	var tasks []models.SyncTaskExtended
	err := database.DB.
		Where("batch_id = ?", batchID).
		Order("batch_index ASC").
		Find(&tasks).Error
	return tasks, err
}

// GetBatchProgress возвращает прогресс выполнения batch
func (s *QueueService) GetBatchProgress(batchID string) (*models.BatchProgress, error) {
	var tasks []models.SyncTaskExtended
	if err := database.DB.Where("batch_id = ?", batchID).Find(&tasks).Error; err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("batch not found")
	}

	progress := &models.BatchProgress{
		BatchID: batchID,
		Total:   len(tasks),
	}

	for _, t := range tasks {
		switch t.Status {
		case models.TaskStatusCompleted:
			progress.Completed++
		case models.TaskStatusFailed, models.TaskStatusCancelled:
			progress.Failed++
		default:
			progress.Pending++
		}
	}

	if progress.Total > 0 {
		progress.Progress = float64(progress.Completed+progress.Failed) / float64(progress.Total) * 100
	}

	return progress, nil
}

// ─── ОБНОВЛЕНИЕ ЗАДАЧ ──────────────────────────────────────

// MarkProcessing помечает задачу как обрабатываемую
func (s *QueueService) MarkProcessing(taskID uint) error {
	now := time.Now()
	result := database.DB.Model(&models.SyncTaskExtended{}).
		Where("id = ? AND status = ?", taskID, models.TaskStatusPending).
		Updates(map[string]interface{}{
			"status":     models.TaskStatusProcessing,
			"started_at": &now,
			"updated_at": now,
		})

	if result.RowsAffected == 0 {
		return fmt.Errorf("task already processing or not found")
	}
	return result.Error
}

// MarkCompleted помечает задачу как выполненную
func (s *QueueService) MarkCompleted(taskID uint, providerReqID int, durationMs int64) error {
	now := time.Now()
	return database.DB.Model(&models.SyncTaskExtended{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":              models.TaskStatusCompleted,
			"completed_at":        &now,
			"duration_ms":         durationMs,
			"provider_request_id": providerReqID,
			"updated_at":          now,
			"last_error":          "",
		}).Error
}

// MarkFailed помечает задачу как неудачную с retry логикой
func (s *QueueService) MarkFailed(task *models.SyncTaskExtended, errMsg string) error {
	task.Attempt++
	task.LastError = errMsg

	updates := map[string]interface{}{
		"attempt":    task.Attempt,
		"last_error": errMsg,
		"updated_at": time.Now(),
	}

	if task.CanRetry() {
		// Планируем повторную попытку
		nextRun := task.CalculateNextRetry()
		updates["status"] = models.TaskStatusPending
		updates["next_run_at"] = &nextRun

		log.Printf("[Queue] Task #%d will retry at %v (attempt %d/%d)",
			task.ID, nextRun.Format("15:04:05"), task.Attempt, task.MaxAttempts)

		// Логируем retry
		Audit.LogQueueRetry(task.ID, task.TargetMSISDN, task.Attempt, task.MaxAttempts, errMsg)
	} else {
		// Исчерпаны попытки
		updates["status"] = models.TaskStatusFailed
		now := time.Now()
		updates["completed_at"] = &now

		log.Printf("[Queue] Task #%d failed permanently after %d attempts: %s",
			task.ID, task.Attempt, errMsg)

		// Логируем провал
		Audit.LogQueueFailed(task.ID, task.TargetMSISDN, errMsg, task.DurationMs)
	}

	return database.DB.Model(&models.SyncTaskExtended{}).Where("id = ?", task.ID).Updates(updates).Error
}

// CancelTask отменяет задачу (только свои задачи)
func (s *QueueService) CancelTask(taskID, userID uint) error {
	result := database.DB.Model(&models.SyncTaskExtended{}).
		Where("id = ? AND user_id = ? AND status IN ?", taskID, userID,
			[]models.TaskStatus{models.TaskStatusPending, models.TaskStatusProcessing}).
		Update("status", models.TaskStatusCancelled)

	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found or cannot be cancelled")
	}
	return result.Error
}

// CancelTaskAdmin отменяет задачу (для админа - любую)
func (s *QueueService) CancelTaskAdmin(taskID uint) error {
	result := database.DB.Model(&models.SyncTaskExtended{}).
		Where("id = ? AND status IN ?", taskID,
			[]models.TaskStatus{models.TaskStatusPending, models.TaskStatusProcessing}).
		Update("status", models.TaskStatusCancelled)

	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found or cannot be cancelled")
	}
	return result.Error
}

// ─── СТАТИСТИКА ────────────────────────────────────────────

// GetStats возвращает статистику очереди
func (s *QueueService) GetStats() (*models.QueueStats, error) {
	stats := &models.QueueStats{}
	today := time.Now().Truncate(24 * time.Hour)

	statuses := map[models.TaskStatus]*int64{
		models.TaskStatusPending:    &stats.Pending,
		models.TaskStatusProcessing: &stats.Processing,
		models.TaskStatusCompleted:  &stats.Completed,
		models.TaskStatusFailed:     &stats.Failed,
		models.TaskStatusCancelled:  &stats.Cancelled,
	}

	for status, counter := range statuses {
		database.DB.Model(&models.SyncTaskExtended{}).Where("status = ?", status).Count(counter)
	}

	database.DB.Model(&models.SyncTaskExtended{}).Where("created_at >= ?", today).Count(&stats.TodayTotal)

	return stats, nil
}

// HasPendingTasks проверяет есть ли ожидающие задачи
func (s *QueueService) HasPendingTasks() bool {
	var count int64
	database.DB.Model(&models.SyncTaskExtended{}).
		Where("status IN ?", []models.TaskStatus{
			models.TaskStatusPending,
			models.TaskStatusProcessing,
		}).
		Count(&count)
	return count > 0
}

// GetPendingCount возвращает количество ожидающих задач
func (s *QueueService) GetPendingCount() int64 {
	var count int64
	database.DB.Model(&models.SyncTaskExtended{}).
		Where("status = ?", models.TaskStatusPending).
		Count(&count)
	return count
}

// ─── ОЧИСТКА ───────────────────────────────────────────────

// CleanupOldTasks удаляет старые завершённые задачи
func (s *QueueService) CleanupOldTasks(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)

	result := database.DB.
		Where("status IN ? AND completed_at < ?",
			[]models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusFailed, models.TaskStatusCancelled},
			cutoff).
		Delete(&models.SyncTaskExtended{})

	if result.Error != nil {
		return 0, result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("[Queue] Cleaned up %d old tasks", result.RowsAffected)
	}

	return result.RowsAffected, nil
}
