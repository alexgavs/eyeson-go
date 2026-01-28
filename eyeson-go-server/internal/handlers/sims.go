// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/models"
	"eyeson-go-server/internal/services"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
	Msisdn    string `json:"msisdn"`
	CLI       string `json:"cli"`
	Field     string `json:"field"`
	Value     string `json:"value"`
	OldValue  string `json:"old_value"`
	RequestID string `json:"request_id,omitempty"`
}

type UpdateSimResponse struct {
	Success   bool   `json:"success"`
	Queued    bool   `json:"queued"`
	TaskID    uint   `json:"task_id,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

func UpdateSim(c *fiber.Ctx) error {
	var req UpdateSimRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UpdateSimResponse{
			Success: false,
			Error:   "Invalid request",
		})
	}

	// Validate
	if req.Msisdn == "" || req.Field == "" {
		return c.Status(fiber.StatusBadRequest).JSON(UpdateSimResponse{
			Success: false,
			Error:   "MSISDN and field are required",
		})
	}

	// Generate request_id if not provided
	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}

	// Get user context
	userCtx := services.Audit.GetUserContext(c)

	// Try direct API call first for label updates
	if req.Field == "label_1" || req.Field == "label_2" || req.Field == "label_3" {
		startTime := time.Now()
		resp, err := eyesont.Instance.UpdateSIMLabel(req.CLI, req.Field, req.Value)
		responseMs := time.Since(startTime).Milliseconds()

		if err == nil && resp != nil && resp.Result == "succeeded" {
			// Success - update local DB
			updateField := "label1"
			if req.Field == "label_2" {
				updateField = "label2"
			} else if req.Field == "label_3" {
				updateField = "label3"
			}
			database.DB.Model(&models.SimCard{}).Where("msisdn = ?", req.Msisdn).Update(updateField, req.Value)

			// Log to audit
			services.Audit.NewLog(c).
				Entity(models.EntitySIM, req.Msisdn).
				Action(models.ActionUpdate).
				Change(req.Field, req.OldValue, req.Value).
				Provider(resp.RequestId, responseMs).
				SaveAsync()

			return c.JSON(UpdateSimResponse{
				Success:   true,
				Queued:    false,
				RequestID: req.RequestID,
			})
		}

		// API failed - queue the task
		log.Printf("[UpdateSim] API call failed, queueing: %v", err)
	}

	// Create queue task
	task, queueErr := services.Queue.CreateTask(services.CreateTaskRequest{
		Type:       models.TaskTypeLabelUpdate,
		Priority:   models.PriorityHigh,
		MSISDN:     req.Msisdn,
		CLI:        req.CLI,
		LabelField: req.Field,
		LabelValue: req.Value,
		UserID:     userCtx.UserID,
		Username:   userCtx.Username,
		IPAddress:  c.IP(),
		RequestID:  req.RequestID,
	})

	if queueErr != nil {
		return c.Status(500).JSON(UpdateSimResponse{
			Success: false,
			Error:   "Failed to queue task: " + queueErr.Error(),
		})
	}

	// Log to audit
	services.Audit.NewLog(c).
		Entity(models.EntitySIM, req.Msisdn).
		Action(models.ActionQueueAdd).
		Change(req.Field, req.OldValue, req.Value).
		Task(task.ID).
		Queued().
		SaveAsync()

	return c.JSON(UpdateSimResponse{
		Success:   true,
		Queued:    true,
		TaskID:    task.ID,
		RequestID: req.RequestID,
	})
}

type BulkStatusRequest struct {
	Status  string              `json:"status"`
	Items   []map[string]string `json:"items"`
	Msisdns []string            `json:"msisdns"`
}

type BulkStatusResponse struct {
	Success     bool   `json:"success"`
	BatchID     string `json:"batch_id"`
	TotalItems  int    `json:"total_items"`
	DirectCount int    `json:"direct_count"`
	QueuedCount int    `json:"queued_count"`
	Error       string `json:"error,omitempty"`
}

// BulkChangeStatus - массовое изменение статуса с поддержкой очереди
func BulkChangeStatus(c *fiber.Ctx) error {
	var req BulkStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(BulkStatusResponse{
			Success: false,
			Error:   "Invalid request",
		})
	}

	// Собираем данные
	type simItem struct {
		MSISDN    string
		CLI       string
		OldStatus string
	}
	var items []simItem

	if len(req.Items) > 0 {
		for _, item := range req.Items {
			items = append(items, simItem{
				MSISDN:    item["msisdn"],
				CLI:       item["cli"],
				OldStatus: item["old_status"],
			})
		}
	} else if len(req.Msisdns) > 0 {
		for _, msisdn := range req.Msisdns {
			items = append(items, simItem{MSISDN: msisdn})
		}
	}

	if len(items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(BulkStatusResponse{
			Success: false,
			Error:   "No SIMs provided",
		})
	}

	userCtx := services.Audit.GetUserContext(c)
	batchID := uuid.New().String()

	// Пытаемся выполнить через API напрямую
	apiItems := make([]map[string]string, 0, len(items))
	for _, item := range items {
		apiItems = append(apiItems, map[string]string{
			"cli":    item.CLI,
			"msisdn": item.MSISDN,
		})
	}

	startTime := time.Now()
	resp, err := eyesont.Instance.BulkUpdateStatus(apiItems, req.Status)
	responseMs := time.Since(startTime).Milliseconds()

	if err == nil && resp != nil && resp.Result == "succeeded" {
		// Успех - обновляем локальную БД
		for _, item := range items {
			database.DB.Model(&models.SimCard{}).Where("msisdn = ? OR cli = ?", item.MSISDN, item.CLI).Update("status", req.Status)

			database.DB.Create(&models.SimHistory{
				MSISDN:    item.MSISDN,
				Action:    "STATUS_CHANGE",
				Field:     "status",
				OldValue:  item.OldStatus,
				NewValue:  req.Status,
				Source:    "USER",
				ChangedBy: userCtx.Username,
			})
		}

		// Логируем bulk операцию
		itemMaps := make([]map[string]string, 0, len(items))
		for _, item := range items {
			itemMaps = append(itemMaps, map[string]string{
				"cli":        item.CLI,
				"msisdn":     item.MSISDN,
				"old_status": item.OldStatus,
			})
		}
		services.Audit.LogBulkStatusChange(c, batchID, itemMaps, req.Status, resp.RequestId, responseMs, nil)

		return c.JSON(BulkStatusResponse{
			Success:     true,
			BatchID:     batchID,
			TotalItems:  len(items),
			DirectCount: len(items),
			QueuedCount: 0,
		})
	}

	// API недоступен - ставим в очередь как batch
	log.Printf("[BulkChangeStatus] API failed, queueing %d items: %v", len(items), err)

	taskRequests := make([]services.CreateTaskRequest, 0, len(items))
	for _, item := range items {
		taskRequests = append(taskRequests, services.CreateTaskRequest{
			Type:      models.TaskTypeStatusChange,
			Priority:  models.PriorityHigh,
			MSISDN:    item.MSISDN,
			CLI:       item.CLI,
			OldStatus: item.OldStatus,
			NewStatus: req.Status,
			UserID:    userCtx.UserID,
			Username:  userCtx.Username,
			IPAddress: c.IP(),
		})
	}

	_, _, queueErr := services.Queue.CreateBatch(taskRequests)
	if queueErr != nil {
		return c.Status(500).JSON(BulkStatusResponse{
			Success: false,
			Error:   "Failed to queue batch: " + queueErr.Error(),
		})
	}

	// Логируем
	itemMaps := make([]map[string]string, 0, len(items))
	for _, item := range items {
		itemMaps = append(itemMaps, map[string]string{
			"cli":        item.CLI,
			"msisdn":     item.MSISDN,
			"old_status": item.OldStatus,
		})
	}
	services.Audit.LogBulkStatusChange(c, batchID, itemMaps, req.Status, 0, responseMs, fmt.Errorf("queued due to provider error"))

	return c.JSON(BulkStatusResponse{
		Success:     true,
		BatchID:     batchID,
		TotalItems:  len(items),
		DirectCount: 0,
		QueuedCount: len(items),
	})
}

// ChangeStatus - изменение статуса одной SIM карты
type ChangeStatusRequest struct {
	CLI       string `json:"cli"`
	MSISDN    string `json:"msisdn"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	RequestID string `json:"request_id,omitempty"`
}

type ChangeStatusResponse struct {
	Success    bool   `json:"success"`
	Queued     bool   `json:"queued"`
	TaskID     uint   `json:"task_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	ProviderID int    `json:"provider_id,omitempty"`
	Error      string `json:"error,omitempty"`
}

func ChangeStatus(c *fiber.Ctx) error {
	var req ChangeStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ChangeStatusResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Валидация
	if req.CLI == "" || req.NewStatus == "" {
		return c.Status(400).JSON(ChangeStatusResponse{
			Success: false,
			Error:   "CLI and new_status are required",
		})
	}

	// Генерируем request_id если не передан
	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}

	userCtx := services.Audit.GetUserContext(c)

	// Пытаемся выполнить запрос к провайдеру
	startTime := time.Now()
	resp, err := eyesont.Instance.UpdateSIMStatus(req.CLI, req.NewStatus)
	responseMs := time.Since(startTime).Milliseconds()

	// Проверяем результат
	if err == nil && resp != nil && resp.Result == "succeeded" {
		// Успех - обновляем локальную БД
		database.DB.Model(&models.SimCard{}).Where("cli = ? OR msisdn = ?", req.CLI, req.MSISDN).Update("status", req.NewStatus)

		// Записываем в историю
		database.DB.Create(&models.SimHistory{
			MSISDN:    req.MSISDN,
			Action:    "STATUS_CHANGE",
			Field:     "status",
			OldValue:  req.OldStatus,
			NewValue:  req.NewStatus,
			Source:    "USER",
			ChangedBy: userCtx.Username,
		})

		// Логируем в аудит
		services.Audit.LogStatusChange(c, req.CLI, req.MSISDN, req.OldStatus, req.NewStatus, resp.RequestId, responseMs, nil)

		return c.JSON(ChangeStatusResponse{
			Success:    true,
			Queued:     false,
			ProviderID: resp.RequestId,
			RequestID:  req.RequestID,
		})
	}

	// Провайдер недоступен - ставим в очередь
	log.Printf("[ChangeStatus] API failed, queueing: %v", err)

	task, queueErr := services.Queue.CreateTask(services.CreateTaskRequest{
		Type:      models.TaskTypeStatusChange,
		Priority:  models.PriorityHigh,
		MSISDN:    req.MSISDN,
		CLI:       req.CLI,
		OldStatus: req.OldStatus,
		NewStatus: req.NewStatus,
		UserID:    userCtx.UserID,
		Username:  userCtx.Username,
		IPAddress: c.IP(),
		RequestID: req.RequestID,
	})

	if queueErr != nil {
		services.Audit.LogStatusChange(c, req.CLI, req.MSISDN, req.OldStatus, req.NewStatus, 0, responseMs, queueErr)
		return c.Status(500).JSON(ChangeStatusResponse{
			Success: false,
			Error:   "Failed to queue operation: " + queueErr.Error(),
		})
	}

	// Логируем постановку в очередь
	services.Audit.LogStatusChangeQueued(c, req.CLI, req.MSISDN, req.OldStatus, req.NewStatus, task.ID, err)

	return c.JSON(ChangeStatusResponse{
		Success:   true,
		Queued:    true,
		TaskID:    task.ID,
		RequestID: req.RequestID,
	})
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
