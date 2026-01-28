// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/models"

	"github.com/gofiber/fiber/v2"
)

// formatTime converts interface{} to string for time fields
func formatTime(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	case int:
		return strconv.Itoa(t)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func GetJobs(c *fiber.Ctx) error {
	start, _ := strconv.Atoi(c.Query("start", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	jobId, _ := strconv.Atoi(c.Query("jobId", "0"))
	jobStatus := c.Query("jobStatus", "")

	log.Printf("[GetJobs] REQUEST: start=%d, limit=%d, jobId=%d, jobStatus='%s'", start, limit, jobId, jobStatus)

	resp, err := eyesont.Instance.GetJobs(start, limit, jobId, jobStatus)
	if err != nil {
		log.Printf("[GetJobs] API ERROR: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Обрабатываем для Frontend, если нужно (сортировка и т.д.)
	// API Pelephone возвращает специфичный формат, возможно нужно адаптировать
	// В данном случае просто проксируем, но можем добавить сортировку по дате

	// Пример сортировки, если API возвращает неотсортировано (обычно API сортирует само)
	if len(resp.Jobs) > 1 {
		sort.SliceStable(resp.Jobs, func(i, j int) bool {
			// Предполагаем, что JobId - автоинкремент
			id1 := resp.Jobs[i].JobId
			id2 := resp.Jobs[j].JobId
			return id1 > id2 // DESC
		})
	}

	log.Printf("[GetJobs] SUCCESS: count=%d", len(resp.Jobs))
	return c.JSON(resp)
}

// GetSyncTasks returns the internal queue status
func GetSyncTasks(c *fiber.Ctx) error {
	var tasks []models.SyncTask

	// Simple query: last 100 tasks
	result := database.DB.Order("created_at desc").Limit(100).Find(&tasks)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": result.Error.Error()})
	}

	// Map to frontend-expected format
	queueTasks := make([]fiber.Map, len(tasks))
	for i, task := range tasks {
		queueTasks[i] = fiber.Map{
			"id":            task.ID,
			"type":          task.Type,
			"payload":       task.Payload,
			"status":        task.Status,
			"created_at":    task.CreatedAt,
			"updated_at":    task.UpdatedAt,
			"attempts":      task.Attempt, // Map 'attempt' to 'attempts'
			"last_error":    task.Result,  // Map 'result' to 'last_error'
			"next_run_at":   task.NextRunAt,
			"target_msisdn": task.TargetMSISDN,
		}
	}

	return c.JSON(fiber.Map{
		"count": len(tasks),
		"data":  queueTasks,
	})
}

// GetLocalJob returns a specific SyncTask formatted as a Job for frontend polling
func GetLocalJob(c *fiber.Ctx) error {
	id := c.Params("id")
	var task models.SyncTask
	if err := database.DB.First(&task, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Task not found"})
	}

	// Map SyncTask status to JobStatus format expected by frontend
	// SyncTask: PENDING, PROCESSING, COMPLETED, FAILED
	// Frontend expects: PENDING, IN_PROGRESS, SUCCESS, FAILED
	jobStatus := task.Status
	if jobStatus == "PROCESSING" {
		jobStatus = "IN_PROGRESS"
	}
	if jobStatus == "COMPLETED" {
		jobStatus = "SUCCESS"
	}

	return c.JSON(fiber.Map{
		"jobStatus": jobStatus, // Direct object return for GetJobStatus compatibility
		"jobId":     task.ID,
		"data": []fiber.Map{ // Array return for API compatibility if needed
			{
				"jobId":          task.ID,
				"jobStatus":      jobStatus,
				"requestTime":    task.CreatedAt,
				"lastActionTime": task.UpdatedAt,
			},
		},
	})
}

// ExecuteQueueTask executes a queued task immediately
func ExecuteQueueTask(c *fiber.Ctx) error {
	taskID := c.Params("id")

	var task models.SyncTask
	if err := database.DB.First(&task, taskID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Task not found"})
	}

	// Check if task can be executed (PENDING or FAILED status)
	if task.Status != "PENDING" && task.Status != "FAILED" {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("Task cannot be executed. Current status: %s", task.Status),
		})
	}

	// Update task to trigger immediate execution
	task.NextRunAt = time.Now().Add(-1 * time.Second) // Set to past to trigger immediately
	task.Status = "PENDING"

	if err := database.DB.Save(&task).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update task"})
	}

	log.Printf("[ExecuteQueueTask] Task #%d scheduled for immediate execution", task.ID)

	return c.JSON(fiber.Map{
		"result":  "success",
		"message": fmt.Sprintf("Task #%d scheduled for immediate execution", task.ID),
	})
}
