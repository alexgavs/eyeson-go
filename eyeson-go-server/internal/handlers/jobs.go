package handlers

import (
	"fmt"
	"log"
	"sort"
	"strconv"

	"eyeson-go-server/internal/eyesont"

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

// GetJobs returns list of provisioning jobs from API
func GetJobs(c *fiber.Ctx) error {
	// Parse pagination params
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 50)
	if perPage > 500 {
		perPage = 500
	}
	start := (page - 1) * perPage

	// Parse filters
	jobIdStr := c.Query("job_id")
	var jobId int
	if jobIdStr != "" {
		if id, err := strconv.Atoi(jobIdStr); err == nil {
			jobId = id
		}
	}
	jobStatus := c.Query("status")

	log.Printf("[GetJobs] REQUEST: page=%d, perPage=%d, jobId=%d, status=%s", page, perPage, jobId, jobStatus)

	// Call API
	resp, err := eyesont.Instance.GetJobs(start, perPage, jobId, jobStatus)
	if err != nil {
		log.Printf("[GetJobs] API ERROR: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Sort jobs by lastActionTime descending (newest first)
	jobs := resp.Jobs
	sort.Slice(jobs, func(i, j int) bool {
		timeI := formatTime(jobs[i].LastActionTime)
		if timeI == "" {
			timeI = formatTime(jobs[i].RequestTime)
		}
		timeJ := formatTime(jobs[j].LastActionTime)
		if timeJ == "" {
			timeJ = formatTime(jobs[j].RequestTime)
		}
		return timeI > timeJ
	})

	total := resp.Count
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	log.Printf("[GetJobs] SUCCESS: count=%d, returned=%d", total, len(jobs))

	return c.JSON(fiber.Map{
		"success": true,
		"data":    jobs,
		"pagination": fiber.Map{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	})
}
