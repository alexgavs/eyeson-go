package handlers

import (
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type StatsResponse struct {
	Total          int64          `json:"total"`
	ByStatus       map[string]int `json:"by_status"`
	ByRatePlan     map[string]int `json:"by_rate_plan"`
	ActiveSessions int64          `json:"active_sessions"`
	LastUpdated    time.Time      `json:"last_updated"`
}

var (
	statsCache      *StatsResponse
	statsCacheMutex sync.Mutex
	statsLastUpdate time.Time
)

// InvalidateStatsCache сбрасывает кэш статистики
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

	// If cache is fresh (e.g. less than 1 min old) and not forced, return it
	if statsCache != nil && !forceRefresh && time.Since(statsLastUpdate) < 1*time.Minute {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    statsCache,
			"cached":  true,
		})
	}

	log.Println("[Stats] Calculating stats from Local DB...")

	stats := &StatsResponse{
		ByStatus:    make(map[string]int),
		ByRatePlan:  make(map[string]int),
		LastUpdated: time.Now(),
	}

	var total int64
	database.DB.Model(&models.SimCard{}).Count(&total)
	stats.Total = total

	// Count by Status
	rows, err := database.DB.Model(&models.SimCard{}).Select("status, count(*)").Group("status").Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var status string
			var count int
			rows.Scan(&status, &count)
			if status == "Active" {
				status = "Activated"
			}
			stats.ByStatus[status] += count
		}
	}

	// Count by RatePlan
	rows2, err := database.DB.Model(&models.SimCard{}).Select("rate_plan, count(*)").Group("rate_plan").Rows()
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var rp string
			var count int
			rows2.Scan(&rp, &count)
			if rp == "" {
				rp = "Unknown"
			}
			stats.ByRatePlan[rp] += count
		}
	}

	// Count Active Sessions
	database.DB.Model(&models.SimCard{}).Where("in_session = ?", true).Count(&stats.ActiveSessions)

	statsCache = stats
	statsLastUpdate = time.Now()

	log.Println("[Stats] Stats calculation completed.")

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
		"cached":  false,
	})
}
