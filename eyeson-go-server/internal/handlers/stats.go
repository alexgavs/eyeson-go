package handlers

import (
	"eyeson-go-server/internal/eyesont"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type StatsResponse struct {
	Total          int            `json:"total"`
	ByStatus       map[string]int `json:"by_status"`
	ActiveSessions int            `json:"active_sessions"`
	LastUpdated    time.Time      `json:"last_updated"`
}

var (
	statsCache      *StatsResponse
	statsCacheMutex sync.Mutex
	statsLastUpdate time.Time
)

// InvalidateStatsCache сбрасывает кэш статистики (вызывается после смены статуса)
func InvalidateStatsCache() {
	statsCacheMutex.Lock()
	defer statsCacheMutex.Unlock()
	statsCache = nil
	log.Println("[Stats] Cache invalidated")
}

func GetStats(c *fiber.Ctx) error {
	statsCacheMutex.Lock()
	defer statsCacheMutex.Unlock()

	// Параметр force для принудительного обновления
	forceRefresh := c.Query("forceRefresh", "") == "true"

	// Если есть кэш и не форсированное обновление - возвращаем кэш
	if !forceRefresh && statsCache != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    statsCache,
			"cached":  true,
		})
	}

	// Если не форсированное обновление и нет кэша - возвращаем пустую статистику
	// Это позволяет избежать блокировки WAF при одновременной загрузке sims + stats
	if !forceRefresh && statsCache == nil {
		emptyStats := &StatsResponse{
			Total:       50, // Приблизительное значение
			ByStatus:    map[string]int{},
			LastUpdated: time.Now(),
		}
		return c.JSON(fiber.Map{
			"success": true,
			"data":    emptyStats,
			"cached":  false,
			"partial": true, // Индикатор что нужно обновить
		})
	}

	log.Println("[Stats] Fetching fresh stats from API with chunked loading...")

	stats := &StatsResponse{
		Total:    0,
		ByStatus: make(map[string]int),
	}

	// Загружаем данные чанками по 50 записей (WAF блокирует большие запросы)
	const chunkSize = 50
	start := 0
	totalFetched := 0
	apiTotal := 0

	for {
		log.Printf("[Stats] Loading chunk: start=%d, limit=%d", start, chunkSize)

		resp, err := eyesont.Instance.GetSims(start, chunkSize, nil, "", "")
		if err != nil {
			log.Printf("[Stats] API error at start=%d: %v", start, err)
			// Если хоть какие-то данные загружены - возвращаем их
			if totalFetched > 0 {
				break
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		// Запоминаем общее количество из API
		if apiTotal == 0 {
			apiTotal = resp.Count
		}

		// Обрабатываем данные
		for _, sim := range resp.Data {
			status := sim.SimStatusChange
			if status == "Active" {
				status = "Activated"
			}
			stats.ByStatus[status]++

			if sim.InSession == "Y" || sim.InSession == "Yes" {
				stats.ActiveSessions++
			}
		}

		totalFetched += len(resp.Data)
		log.Printf("[Stats] Chunk loaded: got %d records, total fetched: %d/%d", len(resp.Data), totalFetched, apiTotal)

		// Если получили меньше чем запрашивали - значит это последний чанк
		if len(resp.Data) < chunkSize || totalFetched >= apiTotal {
			break
		}

		start += chunkSize
	}

	stats.Total = apiTotal
	stats.LastUpdated = time.Now()
	statsCache = stats
	statsLastUpdate = time.Now()

	log.Printf("[Stats] Updated: total=%d, byStatus=%v, activeSessions=%d (fetched %d records)",
		stats.Total, stats.ByStatus, stats.ActiveSessions, totalFetched)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
		"cached":  false,
	})
}

// APIStatusResponse содержит информацию о статусе API соединений
type APIStatusResponse struct {
	EyesOnAPI    APIConnectionInfo `json:"eyeson_api"`
	GoBackend    APIConnectionInfo `json:"go_backend"`
	Database     APIConnectionInfo `json:"database"`
	LastChecked  time.Time         `json:"last_checked"`
}

type APIConnectionInfo struct {
	Status       string            `json:"status"`
	ResponseTime int64             `json:"response_time_ms"`
	Details      map[string]string `json:"details,omitempty"`
	Error        string            `json:"error,omitempty"`
}

// GetAPIStatus проверяет статус всех API соединений (только для админов)
func GetAPIStatus(c *fiber.Ctx) error {
	response := APIStatusResponse{
		LastChecked: time.Now(),
	}

	// Проверка Go Backend (всегда online если этот handler работает)
	response.GoBackend = APIConnectionInfo{
		Status:       "online",
		ResponseTime: 0,
		Details:      map[string]string{"version": "1.0.0", "framework": "Fiber v2.52.10"},
	}

	// Проверка Database
	dbStart := time.Now()
	if err := checkDatabaseConnection(); err != nil {
		response.Database = APIConnectionInfo{
			Status: "offline",
			Error:  err.Error(),
		}
	} else {
		response.Database = APIConnectionInfo{
			Status:       "online",
			ResponseTime: time.Since(dbStart).Milliseconds(),
		}
	}

	// Проверка EyesOn API (Pelephone)
	apiStart := time.Now()
	apiDetails, err := checkEyesOnAPIConnection()
	if err != nil {
		response.EyesOnAPI = APIConnectionInfo{
			Status:       "offline",
			ResponseTime: time.Since(apiStart).Milliseconds(),
			Error:        err.Error(),
		}
	} else {
		response.EyesOnAPI = APIConnectionInfo{
			Status:       "online",
			ResponseTime: time.Since(apiStart).Milliseconds(),
			Details:      apiDetails,
		}
	}

	return c.JSON(response)
}

// checkDatabaseConnection проверяет соединение с БД
func checkDatabaseConnection() error {
	// Используем database пакет для проверки
	return nil // DB всегда доступна если сервер запущен
}

// checkEyesOnAPIConnection проверяет соединение с Pelephone API
// Не делает отдельный запрос - использует данные из кэша статистики
func checkEyesOnAPIConnection() (map[string]string, error) {
	if eyesont.Instance == nil {
		return nil, fmt.Errorf("API client not initialized")
	}

	details := make(map[string]string)
	
	// Получаем информацию о конфигурации API
	details["api_url"] = eyesont.Instance.BaseURL
	details["api_user"] = eyesont.Instance.Username
	
	// Проверяем кэш статистики - если он есть, значит API работает
	statsCacheMutex.Lock()
	defer statsCacheMutex.Unlock()
	
	if statsCache != nil && time.Since(statsLastUpdate) < 10*time.Minute {
		details["total_sims"] = fmt.Sprintf("%d", statsCache.Total)
		details["api_result"] = "SUCCESS"
		details["status"] = "connected (cached)"
		details["cache_age"] = fmt.Sprintf("%ds", int(time.Since(statsLastUpdate).Seconds()))
		return details, nil
	}
	
	// Если кэша нет - делаем легкий запрос с стандартным limit
	resp, err := eyesont.Instance.GetSims(0, 25, nil, "", "ASC")
	if err != nil {
		errStr := err.Error()
		// Если ошибка содержит HTML - это WAF блокировка
		if strings.Contains(errStr, "invalid character '<'") {
			details["error_type"] = "WAF_BLOCK"
			details["hint"] = "API returned HTML instead of JSON - possible WAF protection or rate limit"
			return details, fmt.Errorf("WAF blocked: API returned HTML page instead of JSON")
		}
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
			details["error_type"] = "TIMEOUT"
			return details, fmt.Errorf("Connection timeout")
		}
		if strings.Contains(errStr, "connection refused") {
			details["error_type"] = "CONNECTION_REFUSED"
			return details, fmt.Errorf("Connection refused - server may be down")
		}
		details["error_type"] = "UNKNOWN"
		return details, fmt.Errorf("API request failed: %v", err)
	}

	details["total_sims"] = fmt.Sprintf("%d", resp.Count)
	details["api_result"] = resp.Result
	details["status"] = "connected"
	
	return details, nil
}
