package handlers

import (
	"encoding/json"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/models"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func GetSims(c *fiber.Ctx) error {
	start, _ := strconv.Atoi(c.Query("start", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	searchQuery := c.Query("search", "")
	sortBy := c.Query("sortBy", "")
	sortDirection := c.Query("sortDirection", "ASC")
	statusFilter := c.Query("status", "") // Фильтр по статусу: Activated, Suspended, Terminated

	// Логируем входящий запрос
	log.Printf("[GetSims] REQUEST: start=%d, limit=%d, search='%s', sortBy='%s', sortDirection='%s', status='%s'",
		start, limit, searchQuery, sortBy, sortDirection, statusFilter)

	var searchParams []models.SearchParam
	useApiSearch := false

	// Умный поиск: определяем тип поля по паттерну
	if searchQuery != "" {
		field := ""
		if len(searchQuery) >= 2 && searchQuery[0:2] == "05" {
			field = "CLI"
		} else if len(searchQuery) >= 3 && searchQuery[0:3] == "972" {
			field = "MSISDN"
		} else if len(searchQuery) >= 15 && isNumeric(searchQuery) {
			// IMSI обычно 15 цифр
			field = "IMSI"
		}

		if field != "" {
			useApiSearch = true
			searchParams = append(searchParams, models.SearchParam{
				FieldName:  field,
				FieldValue: searchQuery,
			})
			log.Printf("[GetSims] API SEARCH: field=%s, value=%s", field, searchQuery)
		} else {
			log.Printf("[GetSims] LOCAL SEARCH: query='%s' (will filter locally)", searchQuery)
		}
	}

	// Сортировка - расширяем поддерживаемые поля
	allowedSortFields := map[string]bool{
		"CLI": true, "MSISDN": true, "SIM_STATUS_CHANGE": true,
		"CUSTOMER_LABEL_1": true, "LAST_SESSION_TIME": true,
	}
	if sortBy != "" && !allowedSortFields[sortBy] {
		log.Printf("[GetSims] SORT: field '%s' not allowed, clearing", sortBy)
		sortBy = ""
	}

	// Для локального поиска или фильтра по статусу загружаем больше данных
	var fetchLimit = limit
	var fetchStart = start
	needLocalFilter := (searchQuery != "" && !useApiSearch) || statusFilter != ""
	if needLocalFilter {
		fetchLimit = 5000 // Загружаем больше для локальной фильтрации
		fetchStart = 0    // Начинаем с 0 для полной фильтрации
	}

	log.Printf("[GetSims] CALLING API: fetchStart=%d, fetchLimit=%d, searchParams=%+v, sortBy='%s', sortDirection='%s'",
		fetchStart, fetchLimit, searchParams, sortBy, sortDirection)

	resp, err := eyesont.Instance.GetSims(fetchStart, fetchLimit, searchParams, sortBy, sortDirection)
	if err != nil {
		log.Printf("[GetSims] API ERROR: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	log.Printf("[GetSims] API RESPONSE: count=%d, dataLen=%d", resp.Count, len(resp.Data))

	// Фильтрация по статусу (если указан)
	needStatusFilter := statusFilter != ""
	needLocalSearch := searchQuery != "" && !useApiSearch

	if needStatusFilter || needLocalSearch {
		var filteredData []models.SimData
		queryLower := strings.ToLower(searchQuery)

		for _, sim := range resp.Data {
			// Проверка статуса
			if needStatusFilter && !strings.EqualFold(sim.SimStatusChange, statusFilter) {
				continue
			}

			// Локальный поиск по всем полям
			if needLocalSearch {
				if !strings.Contains(strings.ToLower(sim.CLI), queryLower) &&
					!strings.Contains(strings.ToLower(sim.MSISDN), queryLower) &&
					!strings.Contains(strings.ToLower(sim.CustomerLabel1), queryLower) &&
					!strings.Contains(strings.ToLower(sim.CustomerLabel2), queryLower) &&
					!strings.Contains(strings.ToLower(sim.CustomerLabel3), queryLower) &&
					!strings.Contains(strings.ToLower(sim.SimSwap), queryLower) &&
					!strings.Contains(strings.ToLower(sim.IMSI), queryLower) &&
					!strings.Contains(strings.ToLower(sim.IMEI), queryLower) &&
					!strings.Contains(strings.ToLower(sim.SimStatusChange), queryLower) &&
					!strings.Contains(strings.ToLower(sim.RatePlanFullName), queryLower) &&
					!strings.Contains(strings.ToLower(sim.ApnName), queryLower) &&
					!strings.Contains(strings.ToLower(sim.Ip1), queryLower) {
					continue
				}
			}

			filteredData = append(filteredData, sim)
		}

		totalFiltered := len(filteredData)
		log.Printf("[GetSims] LOCAL FILTER: found %d matches (status='%s', search='%s')", totalFiltered, statusFilter, searchQuery)

		// Применяем пагинацию к отфильтрованным данным
		if start < len(filteredData) {
			end := start + limit
			if end > len(filteredData) {
				end = len(filteredData)
			}
			resp.Data = filteredData[start:end]
		} else {
			resp.Data = []models.SimData{}
		}
		resp.Count = totalFiltered
	}

	log.Printf("[GetSims] FINAL RESPONSE: count=%d, dataLen=%d", resp.Count, len(resp.Data))

	return c.JSON(resp)
}

// isNumeric проверяет, состоит ли строка только из цифр
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

type UpdateSimRequest struct {
	Msisdn string `json:"msisdn"`
	Field  string `json:"field"`
	Value  string `json:"value"`
}

func UpdateSim(c *fiber.Ctx) error {
	var req UpdateSimRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[UpdateSim] PARSE ERROR: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	reqJSON, _ := json.Marshal(req)
	log.Printf("[UpdateSim] REQUEST: %s", string(reqJSON))

	if req.Msisdn == "" || req.Field == "" {
		log.Printf("[UpdateSim] ERROR: Missing fields")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing fields"})
	}

	resp, err := eyesont.Instance.BulkUpdate([]string{req.Msisdn}, req.Field, req.Value)
	if err != nil {
		log.Printf("[UpdateSim] API ERROR: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	log.Printf("[UpdateSim] SUCCESS: msisdn=%s, field=%s, value=%s", req.Msisdn, req.Field, req.Value)

	database.DB.Create(&models.ActivityLog{
		Username:     "admin",
		ActionType:   "update_field",
		TargetMSISDN: req.Msisdn,
		OldValue:     req.Field,
		NewValue:     req.Value,
		Status:       "SUCCESS",
	})

	return c.JSON(resp)
}

type BulkStatusRequest struct {
	Status  string              `json:"status"`
	Items   []map[string]string `json:"items"`
	Msisdns []string            `json:"msisdns"`
}

func BulkChangeStatus(c *fiber.Ctx) error {
	var req BulkStatusRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[BulkChangeStatus] PARSE ERROR: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	reqJSON, _ := json.Marshal(req)
	log.Printf("[BulkChangeStatus] REQUEST: %s", string(reqJSON))

	targetMsisdns := req.Msisdns
	if len(req.Items) > 0 {
		targetMsisdns = []string{}
		for _, item := range req.Items {
			if msisdn, ok := item["msisdn"]; ok {
				targetMsisdns = append(targetMsisdns, msisdn)
			}
		}
	}

	log.Printf("[BulkChangeStatus] Target MSISDNs: %v, Status: %s", targetMsisdns, req.Status)

	if len(targetMsisdns) == 0 {
		log.Printf("[BulkChangeStatus] ERROR: No SIMs provided")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No SIMs provided"})
	}

	resp, err := eyesont.Instance.BulkUpdate(targetMsisdns, "SIM_STATE_CHANGE", req.Status)
	if err != nil {
		log.Printf("[BulkChangeStatus] API ERROR: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	log.Printf("[BulkChangeStatus] SUCCESS: %d SIMs updated to status '%s'", len(targetMsisdns), req.Status)

	// Сбрасываем кэш статистики после успешной смены статуса
	InvalidateStatsCache()

	username := "admin"

	if len(req.Items) > 0 {
		for _, item := range req.Items {
			database.DB.Create(&models.ActivityLog{
				Username:     username,
				ActionType:   "change_status",
				TargetMSISDN: item["msisdn"],
				OldValue:     item["old_status"],
				NewValue:     req.Status,
				Status:       "SUCCESS",
			})
		}
	} else {
		database.DB.Create(&models.ActivityLog{
			Username:   username,
			ActionType: "bulk_change_status",
			NewValue:   req.Status,
			Status:     "SUCCESS",
		})
	}

	return c.JSON(resp)
}
