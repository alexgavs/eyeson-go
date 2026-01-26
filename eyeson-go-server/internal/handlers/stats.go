package handlers

import (
	"eyeson-go-server/internal/eyesont"
	"log"
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
