// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"encoding/json"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func GetSimHistory(c *fiber.Ctx) error {
	msisdn := c.Params("msisdn")
	if msisdn == "" {
		return c.Status(400).JSON(fiber.Map{"error": "MSISDN is required"})
	}

	var history []models.SimHistory
	if err := database.DB.Where("msisdn = ?", msisdn).Order("created_at desc").Find(&history).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch history"})
	}

	// Fetch pending SyncTasks for this MSISDN
	var tasks []models.SyncTask
	database.DB.Where("target_msisdn = ? AND status IN ?", msisdn, []string{"PENDING", "PROCESSING", "QUEUED"}).Find(&tasks)

	// Convert tasks to history-like items to show in UI
	for _, task := range tasks {
		action := "Queued Task"
		val := ""
		
		// Parse payload to get details
		if task.Type == "CHANGE_STATUS" {
			var p struct {
				Status string `json:"status"`
			}
			json.Unmarshal([]byte(task.Payload), &p)
			action = "Status Change (Queued)"
			val = p.Status
		}

		// Create a mock history item
		pendingItem := models.SimHistory{
			MSISDN:   msisdn,
			Action:   action,
			NewValue: val,
			Field:    "status",
			Source:   fmt.Sprintf("System Queue (Att: %d)", task.Attempt),
			CreatedAt: task.CreatedAt,
		}
		
		// Prepend to history
		history = append([]models.SimHistory{pendingItem}, history...)
	}

	return c.JSON(fiber.Map{
		"result": "success",
		"data":   history,
	})
}
