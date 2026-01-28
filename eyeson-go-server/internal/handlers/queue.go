// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"strconv"
	"time"

	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"
	"eyeson-go-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

// ═══════════════════════════════════════════════════════════
// QUEUE HANDLERS
// ═══════════════════════════════════════════════════════════

// GetTaskStatus - получить статус задачи по ID
// GET /api/v1/queue/task/:id
func GetTaskStatus(c *fiber.Ctx) error {
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid task ID"})
	}

	task, err := services.Queue.GetTaskByID(uint(taskID))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Task not found"})
	}

	return c.JSON(task)
}

// GetTaskByRequestID - получить задачу по correlation ID
// GET /api/v1/queue/request/:request_id
func GetTaskByRequestID(c *fiber.Ctx) error {
	requestID := c.Params("request_id")

	if requestID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Request ID is required"})
	}

	task, err := services.Queue.GetTaskByRequestID(requestID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Task not found"})
	}

	return c.JSON(task)
}

// GetMyQueue - получить очередь текущего пользователя
// GET /api/v1/queue/my
func GetMyQueue(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// По умолчанию показываем активные задачи
	tasks, err := services.Queue.GetUserTasks(userID, []models.TaskStatus{
		models.TaskStatusPending,
		models.TaskStatusProcessing,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"items": tasks,
		"count": len(tasks),
	})
}

// GetMyQueueHistory - получить историю задач текущего пользователя
// GET /api/v1/queue/my/history
func GetMyQueueHistory(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Все задачи включая завершённые
	tasks, err := services.Queue.GetUserTasks(userID, nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"items": tasks,
		"count": len(tasks),
	})
}

// GetBatchProgress - получить прогресс batch операции
// GET /api/v1/queue/batch/:batch_id
func GetBatchProgress(c *fiber.Ctx) error {
	batchID := c.Params("batch_id")

	if batchID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Batch ID is required"})
	}

	progress, err := services.Queue.GetBatchProgress(batchID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(progress)
}

// GetBatchTasks - получить все задачи в batch
// GET /api/v1/queue/batch/:batch_id/tasks
func GetBatchTasks(c *fiber.Ctx) error {
	batchID := c.Params("batch_id")

	if batchID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Batch ID is required"})
	}

	tasks, err := services.Queue.GetBatchTasks(batchID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Также получаем прогресс
	progress, _ := services.Queue.GetBatchProgress(batchID)

	return c.JSON(fiber.Map{
		"batch_id": batchID,
		"progress": progress,
		"tasks":    tasks,
	})
}

// CancelTask - отменить свою задачу
// DELETE /api/v1/queue/task/:id
func CancelTask(c *fiber.Ctx) error {
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid task ID"})
	}

	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Получаем задачу для логирования
	task, err := services.Queue.GetTaskByID(uint(taskID))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Task not found"})
	}

	// Отменяем
	if err := services.Queue.CancelTask(uint(taskID), userID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// Логируем отмену
	services.Audit.LogQueueCancel(c, task)

	return c.JSON(fiber.Map{"success": true})
}

// CancelTaskAdmin - отменить любую задачу (admin)
// DELETE /api/v1/queue/admin/task/:id
func CancelTaskAdmin(c *fiber.Ctx) error {
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid task ID"})
	}

	// Получаем задачу для логирования
	task, err := services.Queue.GetTaskByID(uint(taskID))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Task not found"})
	}

	// Отменяем (админ может любую)
	if err := services.Queue.CancelTaskAdmin(uint(taskID)); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// Логируем отмену
	services.Audit.LogQueueCancel(c, task)

	return c.JSON(fiber.Map{"success": true})
}

// GetQueueStats - статистика очереди (admin)
// GET /api/v1/queue/stats
func GetQueueStats(c *fiber.Ctx) error {
	stats, err := services.Queue.GetStats()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(stats)
}

// GetAllPendingTasks - получить все ожидающие задачи (admin)
// GET /api/v1/queue/pending
func GetAllPendingTasks(c *fiber.Ctx) error {
	limit := 100
	if l := c.QueryInt("limit", 100); l > 0 && l <= 500 {
		limit = l
	}

	tasks, err := services.Queue.GetPendingTasks(limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"items": tasks,
		"count": len(tasks),
	})
}

// CleanupOldTasks - очистка старых завершённых задач (admin)
// DELETE /api/v1/queue/cleanup
func CleanupOldTasks(c *fiber.Ctx) error {
	var req struct {
		OlderThanDays int `json:"older_than_days"`
	}
	if err := c.BodyParser(&req); err != nil || req.OlderThanDays < 7 {
		return c.Status(400).JSON(fiber.Map{
			"error": "older_than_days must be at least 7",
		})
	}

	olderThan := 24 * time.Duration(req.OlderThanDays) * time.Hour
	deleted, err := services.Queue.CleanupOldTasks(olderThan)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":       true,
		"deleted_count": deleted,
	})
}

// RetryTask - повторить неудачную задачу (admin)
// POST /api/v1/queue/admin/task/:id/retry
func RetryTask(c *fiber.Ctx) error {
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid task ID"})
	}

	task, err := services.Queue.GetTaskByID(uint(taskID))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Task not found"})
	}

	// Можно повторить только FAILED задачи
	if task.Status != models.TaskStatusFailed {
		return c.Status(400).JSON(fiber.Map{"error": "Can only retry FAILED tasks"})
	}

	// Сбрасываем задачу в PENDING
	now := time.Now()
	result := database.DB.Model(&models.SyncTaskExtended{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":      models.TaskStatusPending,
			"attempt":     0,
			"next_run_at": &now,
			"last_error":  "",
			"updated_at":  now,
		})

	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": result.Error.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "task_id": taskID})
}
