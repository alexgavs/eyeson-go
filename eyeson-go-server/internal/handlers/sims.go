// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"encoding/json"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/models"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Helper to convert DB model to API Response format
func mapModelToApi(m models.SimCard) models.SimData {
	// Format floats
	usage := fmt.Sprintf("%.2f", m.UsageMB)
	allocated := fmt.Sprintf("%.2f", m.AllocatedMB)

	// Format Time
	// API format: 2023-10-27 10:00:00
	lastSession := ""
	if !m.LastSession.IsZero() {
		lastSession = m.LastSession.Format("2006-01-02 15:04:05")
	}

	inSession := "false"
	if m.InSession {
		inSession = "true"
	}

	return models.SimData{
		MSISDN:           m.MSISDN,
		CLI:              m.CLI,
		IMSI:             m.IMSI,
		SimSwap:          m.ICCID,
		IMEI:             m.IMEI,
		SimStatusChange:  m.Status,
		RatePlanFullName: m.RatePlan,
		CustomerLabel1:   m.Label1,
		CustomerLabel2:   m.Label2,
		CustomerLabel3:   m.Label3,
		ApnName:          m.APN,
		Ip1:              m.IP,
		MonthlyUsageMB:   usage,
		AllocatedMB:      allocated,
		LastSessionTime:  lastSession,
		InSession:        inSession,
	}
}

func GetSims(c *fiber.Ctx) error {
	start, _ := strconv.Atoi(c.Query("start", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	searchQuery := c.Query("search", "")
	sortBy := c.Query("sortBy", "")
	sortDirection := c.Query("sortDirection", "ASC")
	statusFilter := c.Query("status", "")

	log.Printf("[GetSims] DB REQUEST: start=%d, limit=%d, search='%s', status='%s'", start, limit, searchQuery, statusFilter)

	db := database.DB.Model(&models.SimCard{})

	// Filter by Status
	if statusFilter != "" {
		db = db.Where("status = ?", statusFilter)
	}

	// Filter by Search Query
	if searchQuery != "" {
		query := "%" + searchQuery + "%"
		db = db.Where("msisdn LIKE ? OR cli LIKE ? OR imsi LIKE ? OR iccid LIKE ? OR label1 LIKE ? OR label2 LIKE ? OR label3 LIKE ?",
			query, query, query, query, query, query, query)
	}

	// Count Total
	var total int64
	db.Count(&total)

	// Sort
	if sortBy != "" {
		// Map API field names to DB columns if necessary
		dbColumn := sortBy
		switch sortBy {
		case "MSISDN":
			dbColumn = "msisdn"
		case "CLI":
			dbColumn = "cli"
		case "SIM_STATUS_CHANGE":
			dbColumn = "status"
		case "CUSTOMER_LABEL_1":
			dbColumn = "label1"
		case "LAST_SESSION_TIME":
			dbColumn = "last_session"
		}

		if sortDirection == "DESC" {
			db = db.Order(dbColumn + " desc")
		} else {
			db = db.Order(dbColumn + " asc")
		}
	} else {
		db = db.Order("updated_at desc")
	}

	// Pagination
	var sims []models.SimCard
	db.Offset(start).Limit(limit).Find(&sims)

	// Check for pending tasks for these SIMs
	var msisdns []string
	for _, s := range sims {
		msisdns = append(msisdns, s.MSISDN)
	}

	pendingTasks := make(map[string]string)
	if len(msisdns) > 0 {
		var tasks []models.SyncTask
		// Check for tasks that are PENDING or PROCESSING
		database.DB.Where("target_msisdn IN ? AND status IN ?", msisdns, []string{"PENDING", "PROCESSING"}).Find(&tasks)
		for _, t := range tasks {
			// We can map the specific type of task if needed
			action := "QUEUED"
			if t.Type == "CHANGE_STATUS" {
				action = "Status Change Queued"
			} else if t.Type == "UPDATE_SIM" {
				action = "Update Queued"
			}
			pendingTasks[t.TargetMSISDN] = action
		}
	}

	// Map to API Response
	var data []models.SimData
	for _, s := range sims {
		apiSim := mapModelToApi(s)
		if status, exists := pendingTasks[s.MSISDN]; exists {
			apiSim.SyncStatus = status
		}
		data = append(data, apiSim)
	}

	// Response
	resp := models.GetProvisioningDataResponse{
		Count:      int(total),
		Data:       data,
		FieldNames: []string{"MSISDN", "CLI", "IMSI", "Status", "RatePlan"}, // Minimal set or full set
	}
	resp.Result = "succeeded" // emulate API success

	return c.JSON(resp)
}

type UpdateSimRequest struct {
	Msisdn string `json:"msisdn"`
	Field  string `json:"field"`
	Value  string `json:"value"`
}

func UpdateSim(c *fiber.Ctx) error {
	var req UpdateSimRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Create Sync Task
	payload, _ := json.Marshal(req)
	task := models.SyncTask{
		Type:         "UPDATE_SIM",
		Payload:      string(payload),
		Status:       "PENDING",
		CreatedBy:    "admin", // TODO: Get from context
		TargetMSISDN: req.Msisdn,
		IPAddress:    c.IP(),
		NextRunAt:    time.Now(),
	}

	if err := database.DB.Create(&task).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to queue task"})
	}

	// Create Audit Log
	database.DB.Create(&models.ActivityLog{
		Username:     "admin",
		ActionType:   "queue_update_field",
		TargetMSISDN: req.Msisdn,
		OldValue:     req.Field,
		NewValue:     req.Value,
		Status:       "QUEUED",
	})

	return c.JSON(fiber.Map{
		"result":  "queued",
		"message": "Update queued for background processing",
		"taskId":  task.ID,
	})
}

type BulkStatusRequest struct {
	Status  string              `json:"status"`
	Items   []map[string]string `json:"items"`
	Msisdns []string            `json:"msisdns"`
}

// Revised Bulk Change that creates individual tasks for tracking
func BulkChangeStatus(c *fiber.Ctx) error {
	var req BulkStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	targetMsisdns := req.Msisdns
	if len(req.Items) > 0 {
		targetMsisdns = []string{}
		for _, item := range req.Items {
			if msisdn, ok := item["msisdn"]; ok {
				targetMsisdns = append(targetMsisdns, msisdn)
			}
		}
	}

	if len(targetMsisdns) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No SIMs provided"})
	}

	// Optimization: Create tasks in batch
	var tasks []models.SyncTask
	now := time.Now()

	for _, msisdn := range targetMsisdns {
		payloadMap := map[string]interface{}{
			"msisdns": []string{msisdn}, // Payload expects list for compatibility
			"status":  req.Status,
		}
		payload, _ := json.Marshal(payloadMap)

		tasks = append(tasks, models.SyncTask{
			Type:         "CHANGE_STATUS",
			Payload:      string(payload),
			Status:       "PENDING",
			CreatedBy:    "admin",
			TargetMSISDN: msisdn,
			IPAddress:    c.IP(),
			NextRunAt:    now,
		})
	}

	if err := database.DB.Create(&tasks).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to queue tasks"})
	}

	response := fiber.Map{
		"result":  "queued",
		"message": fmt.Sprintf("Queued %d status change tasks", len(tasks)),
		"count":   len(tasks),
	}

	if len(tasks) == 1 {
		response["requestId"] = tasks[0].ID
		response["jobId"] = tasks[0].ID
	} else if len(tasks) > 0 {
		// If multiple, maybe return the first one or a list?
		// For now, returning the first one allows tracking at least one.
		// ideally we should return a list.
		response["requestId"] = tasks[0].ID
	}

	return c.JSON(response)
}

func GetAPIStatus(c *fiber.Ctx) error {
	status := "offline"
	message := "Disconnected from EyesOnT API"

	// Real-time check
	if eyesont.Instance != nil {
		if eyesont.Instance.CheckConnection() {
			status = "online"
			message = "Connected to EyesOnT API"
		}
	}

	// Check DB
	dbStatus := "offline"
	if database.DB != nil {
		sqlDB, err := database.DB.DB()
		if err == nil && sqlDB.Ping() == nil {
			dbStatus = "online"
		}
	}

	return c.JSON(models.APIStatusResponse{
		EyesonAPI: models.APIConnectionInfo{
			Status: status,
			Details: map[string]string{
				"api_url":  eyesont.Instance.BaseURL,
				"api_user": eyesont.Instance.Username,
				"message":  message,
			},
		},
		GoBackend:   models.APIConnectionInfo{Status: "online"},
		Database:    models.APIConnectionInfo{Status: dbStatus},
		LastChecked: time.Now().Format(time.RFC3339),
	})
}

type ConnectionRequest struct {
	Action string `json:"action"` // "connect", "disconnect", "set_mode"
	Mode   string `json:"mode"`   // "NORMAL", "REFUSED", "DOWN"
}

// ToggleConnection - This endpoint is deprecated since simulator is now external
// The simulator should be controlled via its own web interface at /web
func ToggleConnection(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "Simulator control moved to external simulator",
		"message": "Use the external simulator's web interface at http://localhost:8888/web to control simulator modes",
	})
}
