// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package jobs

import (
	"encoding/json"
	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/handlers"
	"eyeson-go-server/internal/models"
	"eyeson-go-server/internal/reactive"
	"eyeson-go-server/internal/services"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

type Worker struct {
	DB     *gorm.DB
	Client *eyesont.Client
	paused int32
}

func New(db *gorm.DB) *Worker {
	return &Worker{
		DB:     db,
		Client: eyesont.Instance,
	}
}

func (w *Worker) IsPaused() bool {
	return atomic.LoadInt32(&w.paused) == 1
}

func (w *Worker) Start() {
	log.Println("[JobWorker] Starting background job worker...")

	// Clean up stale tasks on startup
	w.cleanupStaleTasks()

	go func() {
		ticker := time.NewTicker(1 * time.Second) // Check every 1 second for responsiveness
		for range ticker.C {
			if w.IsPaused() {
				continue
			}
			w.ProcessPendingTasks()
		}
	}()
}

// cleanupStaleTasks marks tasks that exceeded max attempts as FAILED
func (w *Worker) cleanupStaleTasks() {
	// Mark tasks stuck in PROCESSING as PENDING (server restart recovery)
	result := w.DB.Model(&models.SyncTaskExtended{}).
		Where("status = ?", "PROCESSING").
		Updates(map[string]interface{}{
			"status":     "PENDING",
			"updated_at": time.Now(),
		})
	if result.RowsAffected > 0 {
		log.Printf("[JobWorker] Recovered %d stuck PROCESSING tasks", result.RowsAffected)
	}

	// Mark tasks that exceeded attempts as FAILED
	// Default MaxAttempts = 3 if not set
	result = w.DB.Model(&models.SyncTaskExtended{}).
		Where("status = ? AND attempt >= CASE WHEN max_attempts = 0 THEN 3 ELSE max_attempts END", "PENDING").
		Updates(map[string]interface{}{
			"status":     "FAILED",
			"result":     "Max attempts exceeded (cleanup)",
			"updated_at": time.Now(),
		})
	if result.RowsAffected > 0 {
		log.Printf("[JobWorker] Marked %d stale tasks as FAILED (exceeded max attempts)", result.RowsAffected)
	}

	// Set max_attempts=3 for tasks where it's 0
	w.DB.Model(&models.SyncTaskExtended{}).
		Where("max_attempts = 0 OR max_attempts IS NULL").
		Update("max_attempts", 3)
}

func (w *Worker) ProcessPendingTasks() {
	var tasks []models.SyncTaskExtended

	// Fetch pending tasks
	if err := w.DB.Where("status = ? AND next_run_at <= ?", "PENDING", time.Now()).Find(&tasks).Error; err != nil {
		log.Printf("[JobWorker] Error fetching tasks: %v", err)
		return
	}

	for _, task := range tasks {
		w.processTask(task)
	}
}

func (w *Worker) processTask(task models.SyncTaskExtended) {
	log.Printf("[JobWorker] Processing task ID=%d Type=%s Target=%s Attempt=%d/%d", task.ID, task.Type, task.TargetMSISDN, task.Attempt, task.MaxAttempts)

	// Ensure MaxAttempts is set (default 3 if not configured)
	if task.MaxAttempts == 0 {
		task.MaxAttempts = 3
		w.DB.Model(&task).Update("max_attempts", 3)
	}

	// Check if already exceeded max attempts - mark as FAILED immediately
	if task.Attempt >= task.MaxAttempts {
		log.Printf("[JobWorker] Task ID=%d exceeded max attempts (%d/%d) - marking as FAILED", task.ID, task.Attempt, task.MaxAttempts)
		w.DB.Model(&task).Updates(map[string]interface{}{
			"status":     "FAILED",
			"result":     fmt.Sprintf("Max attempts exceeded (%d/%d)", task.Attempt, task.MaxAttempts),
			"updated_at": time.Now(),
		})
		services.Audit.LogQueueFailed(task.ID, task.TargetMSISDN, "Max attempts exceeded", 0)
		return
	}

	startTime := time.Now()

	// Update status to PROCESSING
	w.DB.Model(&task).Updates(map[string]interface{}{
		"status":     "PROCESSING",
		"updated_at": time.Now(),
	})

	var err error
	var result string

	switch task.Type {
	case "UPDATE_SIM", "LABEL_UPDATE":
		result, err = w.handleUpdateSim(task)
	case "CHANGE_STATUS", "STATUS_CHANGE", "BULK_CHANGE":
		result, err = w.handleChangeStatus(task)
	default:
		err = fmt.Errorf("unknown task type: %s", task.Type)
	}

	durationMs := time.Since(startTime).Milliseconds()
	status := "COMPLETED"
	if err != nil {
		errMsg := err.Error()
		result = errMsg
		log.Printf("[JobWorker] Task ID=%d FAILED: %v", task.ID, err)

		// Determine Failure Type
		isNetworkError := strings.Contains(errMsg, "connect refused") ||
			strings.Contains(errMsg, "connectex") ||
			strings.Contains(errMsg, "no such host") ||
			strings.Contains(errMsg, "timeout") ||
			strings.Contains(errMsg, "dial tcp")
		isRefused := strings.Contains(errMsg, "500") || strings.Contains(errMsg, "403") || strings.Contains(errMsg, "Server Internal Error")

		// Fatal errors that should NOT be retried - mark as COMPLETED with error
		isFatalError := strings.Contains(errMsg, "not allowed to request_type_id") || // Permission/state issue
			strings.Contains(errMsg, "Permission Denied") ||
			strings.Contains(errMsg, "initial value #null") || // Already in target state
			strings.Contains(errMsg, "subscriber not found") ||
			strings.Contains(errMsg, "invalid subscriber") ||
			strings.Contains(errMsg, "not allowed") ||
			strings.Contains(errMsg, "actionType is mandatory") || // Invalid label update request
			strings.Contains(errMsg, "targetId is mandatory") ||
			strings.Contains(errMsg, "Invalid actionType") ||
			strings.Contains(errMsg, "Invalid targetId") ||
			strings.Contains(errMsg, "System Error") || // Pelephone system error - won't resolve with retries
			(strings.Contains(errMsg, "FAILED") && strings.Contains(errMsg, "target_value")) ||
			(strings.Contains(errMsg, "FAILED") && strings.Contains(errMsg, "mandatory"))

		// Logic based on error type
		if isFatalError {
			log.Printf("[JobWorker] FATAL API Error detected - will not retry. Marking as COMPLETED with error.")
			status = "COMPLETED" // Mark as completed so it doesn't retry

			// Produce a user-friendly result message
			friendlyMsg := errMsg
			if strings.Contains(errMsg, "initial value #null") {
				friendlyMsg = fmt.Sprintf("SIM %s не имеет текущего статуса в системе провайдера (initial value = null). Синхронизируйте данные и попробуйте снова.", task.TargetMSISDN)
			} else if strings.Contains(errMsg, "not allowed to request_type_id") {
				friendlyMsg = fmt.Sprintf("Нет прав на изменение статуса SIM %s. Обратитесь к администратору.", task.TargetMSISDN)
			} else if strings.Contains(errMsg, "subscriber not found") || strings.Contains(errMsg, "invalid subscriber") {
				friendlyMsg = fmt.Sprintf("SIM %s не найдена в системе провайдера.", task.TargetMSISDN)
			}
			result = "SKIPPED: " + friendlyMsg
			// Log to audit
			services.Audit.LogQueueCompleted(task.ID, task.TargetMSISDN, result, durationMs)

			w.DB.Model(&task).Updates(map[string]interface{}{
				"status":     status,
				"result":     result,
				"updated_at": time.Now(),
			})
			return
		} else if isNetworkError {
			log.Printf("[JobWorker] Network Error detected. Server might be DOWN.")
		} else if isRefused {
			log.Printf("[JobWorker] Server REFUSED the request.")
		}

		// Retry logic for non-fatal errors
		if task.Attempt < task.MaxAttempts {
			backoffMinutes := task.Attempt + 1
			if isNetworkError {
				backoffMinutes = (task.Attempt + 1) * 2 // Slower backoff for network issues
			}

			nextRetry := time.Now().Add(time.Minute * time.Duration(backoffMinutes))
			w.DB.Model(&task).Updates(map[string]interface{}{
				"status":      "PENDING", // Back to pending
				"attempt":     task.Attempt + 1,
				"next_run_at": nextRetry,
				"result":      "RETRYING: " + result,
			})

			// Log retry to audit
			services.Audit.LogQueueRetry(task.ID, task.TargetMSISDN, task.Attempt+1, task.MaxAttempts, errMsg)
			return
		} else {
			status = "FAILED"
			// Log failure to audit
			services.Audit.LogQueueFailed(task.ID, task.TargetMSISDN, errMsg, durationMs)
		}
	} else {
		log.Printf("[JobWorker] Task ID=%d COMPLETED", task.ID)
		// Log completion to audit
		services.Audit.LogQueueCompleted(task.ID, task.TargetMSISDN, result, durationMs)

		// Invalidate stats cache on successful status/label change
		if task.Type == "CHANGE_STATUS" || task.Type == "STATUS_CHANGE" || task.Type == "BULK_CHANGE" {
			handlers.InvalidateStatsCache()
		}
	}

	w.DB.Model(&task).Updates(map[string]interface{}{
		"status":     status,
		"result":     result,
		"updated_at": time.Now(),
	})

	// Broadcast SSE event to notify UI of task completion/failure
	broadcaster := handlers.GetEventBroadcaster()
	if broadcaster != nil {
		eventData := map[string]interface{}{
			"task_id":       task.ID,
			"type":          task.Type,
			"status":        status,
			"msisdn":        task.TargetMSISDN,
			"result":        result,
			"duration_ms":   durationMs,
			"label_field":   task.LabelField,
			"label_value":   task.LabelValue,
		}

		if status == "COMPLETED" {
			broadcaster.Emit(reactive.EventTaskCompleted, eventData, "")
			log.Printf("[JobWorker] Broadcast TASK_COMPLETED event for task %d", task.ID)
		} else if status == "FAILED" {
			broadcaster.Emit(reactive.EventTaskFailed, eventData, "")
			log.Printf("[JobWorker] Broadcast TASK_FAILED event for task %d", task.ID)
		}
	}

	// Create History Log for final status
	if status == "FAILED" || status == "COMPLETED" {
		w.DB.Create(&models.SimHistory{
			MSISDN:   task.TargetMSISDN,
			Action:   "TASK_" + status,
			Field:    "status", // generic
			NewValue: status,
			Source:   "WORKER",
			TaskID:   &task.ID,
			OldValue: result, // Store error or result msg
		})
	}
}

// Payload structs matching handlers
type UpdateSimPayload struct {
	Msisdn string `json:"msisdn"`
	Field  string `json:"field"`
	Value  string `json:"value"`
}

func (w *Worker) handleUpdateSim(task models.SyncTaskExtended) (string, error) {
	var msisdn, field, value string

	// Для LABEL_UPDATE используем поля LabelField и LabelValue из задачи
	if task.Type == models.TaskTypeLabelUpdate && task.LabelField != "" {
		msisdn = task.TargetMSISDN
		if msisdn == "" {
			msisdn = task.TargetCLI
		}
		field = task.LabelField
		value = task.LabelValue

		// Нормализуем поле (CUSTOMER_LABEL_1 -> label_1)
		switch field {
		case "CUSTOMER_LABEL_1":
			field = "label_1"
		case "CUSTOMER_LABEL_2":
			field = "label_2"
		case "CUSTOMER_LABEL_3":
			field = "label_3"
		}

		log.Printf("[Worker] LABEL_UPDATE: msisdn=%s, field=%s, value=%s", msisdn, field, value)
	} else {
		// Fallback для UPDATE_SIM - парсим из Payload
		var p UpdateSimPayload
		if err := json.Unmarshal([]byte(task.Payload), &p); err != nil {
			return "", fmt.Errorf("failed to parse payload: %w", err)
		}
		msisdn = p.Msisdn
		field = p.Field
		value = p.Value
	}

	if msisdn == "" {
		return "", fmt.Errorf("msisdn is required")
	}

	// Для label updates используем UpdateSIMLabel
	var resp *models.BulkUpdateResponse
	var err error
	if field == "label_1" || field == "label_2" || field == "label_3" {
		resp, err = w.Client.UpdateSIMLabel(msisdn, field, value)
	} else {
		resp, err = w.Client.BulkUpdate([]string{msisdn}, field, value)
	}

	if err != nil {
		return "", err
	}

	if resp != nil && resp.Result != "succeeded" && resp.Result != "SUCCESS" {
		return "", fmt.Errorf("API Error: %s - %s", resp.Result, resp.Message)
	}

	// Update local DB to reflect change immediately
	if field == "label_1" || field == "label_2" || field == "label_3" {
		dbField := "label1"
		if field == "label_2" {
			dbField = "label2"
		} else if field == "label_3" {
			dbField = "label3"
		}
		w.DB.Model(&models.SimCard{}).Where("msisdn = ? OR cli = ?", msisdn, msisdn).Update(dbField, value)
	}

	// Create History
	w.DB.Create(&models.SimHistory{
		SimID:    0, // We need to lookup ID, skipped for now
		MSISDN:   msisdn,
		Action:   "UPDATE_FIELD",
		Field:    field,
		NewValue: value,
		Source:   "SYNC_WORKER",
		TaskID:   &task.ID,
	})

	// НЕ синхронизируем с API сразу - Pelephone имеет eventual consistency
	// Запланируем отложенную синхронизацию через 15 секунд (увеличено с 5 до 15 для избежания race condition)
	go func(m string) {
		time.Sleep(15 * time.Second)
		w.syncSimsFromAPI([]string{m})
	}(msisdn)

	return "Update successful", nil
}

type BulkStatusPayload struct {
	Msisdns []string `json:"msisdns"`
	Status  string   `json:"status"`
}

func (w *Worker) handleChangeStatus(task models.SyncTaskExtended) (string, error) {
	var p BulkStatusPayload
	if err := json.Unmarshal([]byte(task.Payload), &p); err != nil {
		return "", err
	}

	if len(p.Msisdns) == 0 {
		return "No MSISDNs", nil
	}

	// Pre-validate: query each SIM's current status from upstream API.
	// The Pelephone API determines "initialValue" server-side; if it resolves
	// to null the request is rejected. By checking beforehand we can give a
	// clear error instead of a cryptic permission-denied message.
	for _, msisdn := range p.Msisdns {
		upstreamStatus, err := w.Client.GetSimStatus(msisdn)
		if err != nil {
			log.Printf("[JobWorker] Pre-validation WARNING for %s: %v", msisdn, err)
			// Network errors should not block – let the actual API call handle it
			continue
		}
		if upstreamStatus == "" {
			return "", fmt.Errorf(
				"SIM %s has no status in upstream API (initial value would be null). "+
					"Cannot change status to %s. Please sync data first or verify the SIM in the provider portal",
				eyesont.NormalizeMSISDN(msisdn), p.Status)
		}
		log.Printf("[JobWorker] Pre-validated SIM %s: current upstream status = %s", msisdn, upstreamStatus)
	}

	// Call API
	resp, err := w.Client.BulkUpdate(p.Msisdns, "SIM_STATE_CHANGE", p.Status)
	if err != nil {
		return "", err
	}

	if resp != nil && resp.Result != "succeeded" && resp.Result != "SUCCESS" {
		return "", fmt.Errorf("API Error: %s - %s", resp.Result, resp.Message)
	}

	// Fetch old statuses before update
	var oldSims []models.SimCard
	w.DB.Where("msisdn IN ?", p.Msisdns).Find(&oldSims)
	oldStatusMap := make(map[string]string)
	for _, sim := range oldSims {
		oldStatusMap[sim.MSISDN] = sim.Status
	}

	// Update local DB for immediate UI feedback
	w.DB.Model(&models.SimCard{}).Where("msisdn IN ?", p.Msisdns).Update("status", p.Status)

	// НЕ синхронизируем с API сразу - Pelephone имеет eventual consistency
	// API вернёт старый статус в течение 2-5 секунд после обновления
	// Синхронизация произойдёт при следующем полном sync цикле
	// w.syncSimsFromAPI(p.Msisdns)

	// Запланируем отложенную синхронизацию через 15 секунд (увеличено с 5 до 15 для избежания race condition)
	go func(msisdns []string) {
		time.Sleep(15 * time.Second)
		w.syncSimsFromAPI(msisdns)
		handlers.InvalidateStatsCache()
	}(p.Msisdns)

	// Create history records for each SIM
	for _, msisdn := range p.Msisdns {
		oldStatus := oldStatusMap[msisdn]
		if oldStatus == "" {
			oldStatus = "Unknown"
		}

		w.DB.Create(&models.SimHistory{
			MSISDN:    msisdn,
			Action:    "STATUS_CHANGE",
			Field:     "status",
			OldValue:  oldStatus,
			NewValue:  p.Status,
			Source:    "WORKER",
			ChangedBy: "system",
			TaskID:    &task.ID,
		})
	}

	return fmt.Sprintf("Updated %d SIMs", len(p.Msisdns)), nil
}

// syncSimsFromAPI fetches and updates SIM data from API after task completion
func (w *Worker) syncSimsFromAPI(msisdns []string) {
	if w.Client == nil || len(msisdns) == 0 {
		return
	}

	log.Printf("[JobWorker] Syncing %d SIMs from API after task completion", len(msisdns))

	// Fetch updated data from API for each MSISDN
	for _, msisdn := range msisdns {
		// Build search criteria for this specific SIM
		searchCriteria := []models.SearchParam{
			{
				FieldName:  "MSISDN",
				FieldValue: msisdn,
			},
		}

		// Fetch from API
		resp, err := w.Client.GetSims(0, 1, searchCriteria, "", "")
		if err != nil {
			log.Printf("[JobWorker] Failed to sync SIM %s: %v", msisdn, err)
			continue
		}

		if len(resp.Data) == 0 {
			log.Printf("[JobWorker] SIM %s not found in API response", msisdn)
			continue
		}

		simData := resp.Data[0]

		// Update database with fresh data from API
		var sim models.SimCard
		result := w.DB.Where("msisdn = ?", msisdn).First(&sim)

		if result.Error != nil {
			// SIM doesn't exist, create it
			sim = models.SimCard{
				MSISDN: msisdn,
			}
		}

		// Update all fields from API
		sim.CLI = simData.CLI
		sim.IMSI = simData.IMSI
		sim.ICCID = simData.SimSwap
		sim.IMEI = simData.IMEI
		sim.Status = simData.SimStatusChange
		sim.RatePlan = simData.RatePlanFullName
		sim.Label1 = simData.CustomerLabel1
		sim.Label2 = simData.CustomerLabel2
		sim.Label3 = simData.CustomerLabel3
		sim.APN = simData.ApnName
		sim.IP = simData.Ip1
		sim.InSession = simData.InSession == "true"

		// Save to DB
		if result.Error != nil {
			w.DB.Create(&sim)
		} else {
			w.DB.Save(&sim)
		}

		log.Printf("[JobWorker] ✅ Synced SIM %s from API", msisdn)
	}
}
