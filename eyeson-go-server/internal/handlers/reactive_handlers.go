// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System - Reactive Handlers

package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"
	"eyeson-go-server/internal/reactive"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

var (
	eventBroadcaster *reactive.EventBroadcaster
	simRepository    *reactive.SimRepository
)

func init() {
	eventBroadcaster = reactive.NewEventBroadcaster()
	simRepository = reactive.NewSimRepository()
}

// GetEventBroadcaster returns the global event broadcaster instance
func GetEventBroadcaster() *reactive.EventBroadcaster {
	return eventBroadcaster
}

// ReactiveEventsHandler establishes a reactive SSE connection
func ReactiveEventsHandler(c *fiber.Ctx) error {
	userID := c.Query("user_id", "")
	_ = c.Query("types", "") // TODO: implement type filtering

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		log.Printf("[ReactiveSSE] Client connected (User: %s)", userID)
		defer log.Println("[ReactiveSSE] Client disconnected")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Send welcome event
		welcomeData, _ := json.Marshal(map[string]interface{}{
			"message": "Reactive SSE connection established",
			"time":    time.Now(),
			"user_id": userID,
		})
		fmt.Fprintf(w, "data: %s\n\n", welcomeData)
		w.Flush()

		// Subscribe to broadcaster - get a dedicated channel for this client
		subCh := eventBroadcaster.Subscribe()
		defer eventBroadcaster.Unsubscribe(subCh)

		// Keep-alive ticker
		keepAliveTicker := time.NewTicker(15 * time.Second)
		defer keepAliveTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-subCh:
				if !ok {
					return
				}

				// Filter by user if specified
				if userID != "" && event.UserID != userID && event.UserID != "" {
					continue
				}

				jsonData, err := json.Marshal(event)
				if err != nil {
					log.Printf("[ReactiveSSE] Marshal error: %v", err)
					continue
				}

				fmt.Fprintf(w, "data: %s\n\n", jsonData)
				if err := w.Flush(); err != nil {
					log.Printf("[ReactiveSSE] Flush error: %v", err)
					return
				}

			case <-keepAliveTicker.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}

// ReactiveSimsListHandler returns SIMs as a reactive stream
func ReactiveSimsListHandler(c *fiber.Ctx) error {
	ctx := context.Background()

	var sims []interface{}

	// Get all SIMs as stream
	simStream := simRepository.GetAllAsStream(ctx)

	// Collect to array
	for item := range simStream.ToChannel() {
		if item.Error() {
			return c.Status(500).JSON(fiber.Map{
				"error": item.E.Error(),
			})
		}
		sims = append(sims, item.V)
	}

	return c.JSON(fiber.Map{
		"sims":  sims,
		"count": len(sims),
	})
}

// ReactiveSimSearchHandler performs search via direct DB query.
// Supports field-specific search with "field:query" syntax (e.g. "msisdn:972")
// and general search across ICCID, MSISDN, IMSI, CLI, Status, RatePlan.
func ReactiveSimSearchHandler(c *fiber.Ctx) error {
	raw := c.Query("q", "")

	if raw == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Search query is required",
		})
	}

	// Parse field-specific search: "field:value"
	var field, query string
	if idx := strings.Index(raw, ":"); idx > 0 && idx < len(raw)-1 {
		field = strings.ToLower(raw[:idx])
		query = raw[idx+1:]
	} else {
		field = "all"
		query = raw
	}

	// Map field names → GORM column names
	ns := database.DB.Config.NamingStrategy
	fieldMap := map[string]string{
		"iccid":     ns.ColumnName("", "ICCID"),
		"msisdn":    ns.ColumnName("", "MSISDN"),
		"imsi":      ns.ColumnName("", "IMSI"),
		"cli":       ns.ColumnName("", "CLI"),
		"status":    ns.ColumnName("", "Status"),
		"rate_plan": ns.ColumnName("", "RatePlan"),
	}

	var sims []models.SimCard
	like := "%" + query + "%"

	if field != "all" {
		colName, ok := fieldMap[field]
		if !ok {
			return c.Status(400).JSON(fiber.Map{
				"error":        "Unknown search field: " + field,
				"valid_fields": []string{"iccid", "msisdn", "imsi", "cli", "status", "rate_plan"},
			})
		}
		if err := database.DB.Where(
			fmt.Sprintf("%s LIKE ?", colName), like,
		).Limit(100).Find(&sims).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	} else {
		// Search across all indexed fields
		cols := make([]string, 0, len(fieldMap))
		args := make([]interface{}, 0, len(fieldMap))
		for _, col := range fieldMap {
			cols = append(cols, fmt.Sprintf("%s LIKE ?", col))
			args = append(args, like)
		}
		if err := database.DB.Where(
			strings.Join(cols, " OR "), args...,
		).Limit(100).Find(&sims).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.JSON(fiber.Map{
		"results": sims,
		"count":   len(sims),
		"query":   query,
		"field":   field,
	})
}

// ReactiveStatsHandler returns aggregated event statistics
func ReactiveStatsHandler(c *fiber.Ctx) error {
	stats := eventBroadcaster.GetStats()
	return c.JSON(stats)
}

// EmitSimEvent is a helper to emit SIM-related events
func EmitSimEvent(eventType reactive.EventType, sim interface{}, userID string) {
	eventBroadcaster.Emit(eventType, sim, userID)
}
