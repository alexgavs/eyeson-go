// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"fmt"
	"strings"
	"time"

	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"
	"eyeson-go-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

// ═══════════════════════════════════════════════════════════
// AUDIT FILTER
// ═══════════════════════════════════════════════════════════

// AuditFilter - фильтры для запроса аудита
type AuditFilter struct {
	Page       int    `query:"page"`
	Limit      int    `query:"limit"`
	Username   string `query:"username"`
	EntityType string `query:"entity_type"`
	EntityID   string `query:"entity_id"`
	Action     string `query:"action"`
	Status     string `query:"status"`
	Source     string `query:"source"`
	DateFrom   string `query:"date_from"`
	DateTo     string `query:"date_to"`
	BatchID    string `query:"batch_id"`
}

// ═══════════════════════════════════════════════════════════
// AUDIT HANDLERS
// ═══════════════════════════════════════════════════════════

// GetAuditLogs - получить логи аудита с фильтрацией и пагинацией
// GET /api/v1/audit
func GetAuditLogs(c *fiber.Ctx) error {
	filter := AuditFilter{
		Page:  1,
		Limit: 50,
	}
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid query parameters"})
	}

	// Валидация пагинации
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 50
	}

	offset := (filter.Page - 1) * filter.Limit
	query := database.DB.Model(&models.AuditLog{})

	// ─── ПРИМЕНЯЕМ ФИЛЬТРЫ ─────────────────────────────────
	if filter.Username != "" {
		query = query.Where("username LIKE ?", "%"+filter.Username+"%")
	}
	if filter.EntityType != "" {
		query = query.Where("entity_type = ?", filter.EntityType)
	}
	if filter.EntityID != "" {
		query = query.Where("entity_id LIKE ?", "%"+filter.EntityID+"%")
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}
	if filter.BatchID != "" {
		query = query.Where("batch_id = ?", filter.BatchID)
	}
	if filter.DateFrom != "" {
		if t, err := time.Parse("2006-01-02", filter.DateFrom); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if filter.DateTo != "" {
		if t, err := time.Parse("2006-01-02", filter.DateTo); err == nil {
			query = query.Where("created_at < ?", t.Add(24*time.Hour))
		}
	}

	// Подсчёт общего количества
	var total int64
	query.Count(&total)

	// Получаем записи с пагинацией
	var logs []models.AuditLog
	query.Order("created_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&logs)

	return c.JSON(fiber.Map{
		"data":        logs,
		"total":       total,
		"page":        filter.Page,
		"limit":       filter.Limit,
		"total_pages": (total + int64(filter.Limit) - 1) / int64(filter.Limit),
	})
}

// GetEntityHistory - история изменений конкретной сущности
// GET /api/v1/audit/entity/:type/:id
func GetEntityHistory(c *fiber.Ctx) error {
	entityType := c.Params("type")
	entityID := c.Params("id")

	if entityType == "" || entityID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Entity type and ID are required"})
	}

	var logs []models.AuditLog
	database.DB.
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Limit(100).
		Find(&logs)

	return c.JSON(fiber.Map{
		"entity_type": entityType,
		"entity_id":   entityID,
		"history":     logs,
		"count":       len(logs),
	})
}

// GetSIMHistory - история изменений SIM карты по MSISDN
// GET /api/v1/audit/sim/:msisdn
func GetSIMHistory(c *fiber.Ctx) error {
	msisdn := c.Params("msisdn")

	if msisdn == "" {
		return c.Status(400).JSON(fiber.Map{"error": "MSISDN is required"})
	}

	var logs []models.AuditLog
	database.DB.
		Where("entity_type = ? AND entity_id = ?", models.EntitySIM, msisdn).
		Order("created_at DESC").
		Limit(100).
		Find(&logs)

	return c.JSON(fiber.Map{
		"msisdn":  msisdn,
		"history": logs,
		"count":   len(logs),
	})
}

// GetUserActivity - активность конкретного пользователя
// GET /api/v1/audit/user/:user_id
func GetUserActivity(c *fiber.Ctx) error {
	userID := c.Params("user_id")

	if userID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "User ID is required"})
	}

	var logs []models.AuditLog
	database.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(100).
		Find(&logs)

	return c.JSON(fiber.Map{
		"user_id":  userID,
		"activity": logs,
		"count":    len(logs),
	})
}

// GetAuditStats - статистика аудита
// GET /api/v1/audit/stats
func GetAuditStats(c *fiber.Ctx) error {
	today := time.Now().Truncate(24 * time.Hour)
	week := today.AddDate(0, 0, -7)
	month := today.AddDate(0, -1, 0)

	stats := make(map[string]interface{})

	// Общая статистика
	var totalActions, todayActions, weekActions, monthActions, failedActions int64
	database.DB.Model(&models.AuditLog{}).Count(&totalActions)
	database.DB.Model(&models.AuditLog{}).Where("created_at >= ?", today).Count(&todayActions)
	database.DB.Model(&models.AuditLog{}).Where("created_at >= ?", week).Count(&weekActions)
	database.DB.Model(&models.AuditLog{}).Where("created_at >= ?", month).Count(&monthActions)
	database.DB.Model(&models.AuditLog{}).Where("status = ?", models.AuditStatusFailed).Count(&failedActions)

	stats["total_actions"] = totalActions
	stats["today_actions"] = todayActions
	stats["week_actions"] = weekActions
	stats["month_actions"] = monthActions
	stats["failed_actions"] = failedActions

	// Топ пользователей за неделю
	type UserCount struct {
		Username string `json:"username"`
		Count    int64  `json:"count"`
	}
	var topUsers []UserCount
	database.DB.Model(&models.AuditLog{}).
		Select("username, count(*) as count").
		Where("created_at >= ? AND username != 'system' AND username != 'worker'", week).
		Group("username").
		Order("count DESC").
		Limit(5).
		Scan(&topUsers)
	stats["top_users"] = topUsers

	// Распределение по действиям за неделю
	type ActionCount struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}
	var actionDist []ActionCount
	database.DB.Model(&models.AuditLog{}).
		Select("action, count(*) as count").
		Where("created_at >= ?", week).
		Group("action").
		Order("count DESC").
		Scan(&actionDist)
	stats["action_distribution"] = actionDist

	// Распределение по источникам за неделю
	type SourceCount struct {
		Source string `json:"source"`
		Count  int64  `json:"count"`
	}
	var sourceDist []SourceCount
	database.DB.Model(&models.AuditLog{}).
		Select("source, count(*) as count").
		Where("created_at >= ?", week).
		Group("source").
		Order("count DESC").
		Scan(&sourceDist)
	stats["source_distribution"] = sourceDist

	// Статусы операций за неделю
	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusDist []StatusCount
	database.DB.Model(&models.AuditLog{}).
		Select("status, count(*) as count").
		Where("created_at >= ?", week).
		Group("status").
		Order("count DESC").
		Scan(&statusDist)
	stats["status_distribution"] = statusDist

	// Активность по дням за последние 7 дней
	type DailyCount struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	var dailyActivity []DailyCount
	database.DB.Model(&models.AuditLog{}).
		Select("DATE(created_at) as date, count(*) as count").
		Where("created_at >= ?", week).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&dailyActivity)
	stats["daily_activity"] = dailyActivity

	return c.JSON(stats)
}

// ExportAuditLogs - экспорт логов в CSV
// GET /api/v1/audit/export
func ExportAuditLogs(c *fiber.Ctx) error {
	filter := AuditFilter{}
	c.QueryParser(&filter)

	query := database.DB.Model(&models.AuditLog{})

	// Применяем те же фильтры
	if filter.Username != "" {
		query = query.Where("username LIKE ?", "%"+filter.Username+"%")
	}
	if filter.EntityType != "" {
		query = query.Where("entity_type = ?", filter.EntityType)
	}
	if filter.EntityID != "" {
		query = query.Where("entity_id LIKE ?", "%"+filter.EntityID+"%")
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}
	if filter.DateFrom != "" {
		if t, err := time.Parse("2006-01-02", filter.DateFrom); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if filter.DateTo != "" {
		if t, err := time.Parse("2006-01-02", filter.DateTo); err == nil {
			query = query.Where("created_at < ?", t.Add(24*time.Hour))
		}
	}

	var logs []models.AuditLog
	query.Order("created_at DESC").Limit(10000).Find(&logs)

	// Логируем экспорт
	services.Audit.LogExport(c, "audit_logs", len(logs))

	// Генерируем CSV
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", "attachment; filename=audit_log_"+time.Now().Format("2006-01-02")+".csv")

	// BOM для UTF-8
	csv := "\xEF\xBB\xBF"
	// Header
	csv += "ID,Created At,Username,User Role,Entity Type,Entity ID,Action,Field,Old Value,New Value,Status,Error,Source,IP Address,Provider Request ID\n"

	for _, log := range logs {
		csv += fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%d\n",
			log.ID,
			log.CreatedAt.Format("2006-01-02 15:04:05"),
			escapeCSV(log.Username),
			escapeCSV(log.UserRole),
			log.EntityType,
			escapeCSV(log.EntityID),
			log.Action,
			escapeCSV(log.Field),
			escapeCSV(log.OldValue),
			escapeCSV(log.NewValue),
			log.Status,
			escapeCSV(log.ErrorMessage),
			log.Source,
			log.IPAddress,
			log.ProviderRequestID,
		)
	}

	return c.SendString(csv)
}

// escapeCSV экранирует значение для CSV
func escapeCSV(s string) string {
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// CleanupOldAuditLogs - очистка старых логов (admin)
// DELETE /api/v1/audit/cleanup
func CleanupOldAuditLogs(c *fiber.Ctx) error {
	var req struct {
		OlderThanDays int `json:"older_than_days"`
	}
	if err := c.BodyParser(&req); err != nil || req.OlderThanDays < 30 {
		return c.Status(400).JSON(fiber.Map{
			"error": "older_than_days must be at least 30",
		})
	}

	cutoff := time.Now().AddDate(0, 0, -req.OlderThanDays)

	result := database.DB.
		Where("created_at < ?", cutoff).
		Delete(&models.AuditLog{})

	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": result.Error.Error()})
	}

	return c.JSON(fiber.Map{
		"success":       true,
		"deleted_count": result.RowsAffected,
		"cutoff_date":   cutoff.Format("2006-01-02"),
	})
}
