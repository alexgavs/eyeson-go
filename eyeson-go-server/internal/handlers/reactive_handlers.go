// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System - Reactive Handlers

package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"eyeson-go-server/internal/reactive"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/reactivex/rxgo/v2"
	"github.com/valyala/fasthttp"
)

var (
	eventBroadcaster *reactive.EventBroadcaster
	simRepository    *reactive.SimRepository
)

func init() {
	eventBroadcaster = reactive.NewEventBroadcaster()
	simRepository = reactive.NewSimRepository()

	// Start event logging
	go eventBroadcaster.LogEvents(context.Background())
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

		// Create SSE stream with filters
		var sseStream *reactive.Stream
		if userID != "" {
			// Filter by user and convert to SSE
			sseStream = eventBroadcaster.ToSSE(ctx, func(s *reactive.Stream) *reactive.Stream {
				return s.Filter(func(item interface{}) bool {
					event, ok := item.(reactive.Event)
					if !ok {
						return false
					}
					return event.UserID == userID || event.UserID == ""
				})
			})
		} else {
			sseStream = eventBroadcaster.ToSSE(ctx)
		}

		// Keep-alive ticker
		keepAliveTicker := time.NewTicker(15 * time.Second)
		defer keepAliveTicker.Stop()

		// Subscribe to events
		eventCh := sseStream.ToChannel()

		for {
			select {
			case <-ctx.Done():
				return

			case item, ok := <-eventCh:
				if !ok {
					return
				}

				if item.Error() {
					log.Printf("[ReactiveSSE] Error: %v", item.E)
					continue
				}

				jsonData, ok := item.V.([]byte)
				if !ok {
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

// ReactiveSimSearchHandler performs debounced search
func ReactiveSimSearchHandler(c *fiber.Ctx) error {
	query := c.Query("q", "")

	if query == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Search query is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create search stream
	searchCh := make(chan rxgo.Item, 1)
	searchCh <- rxgo.Of(query)
	close(searchCh)

	var results []interface{}

	// Execute search
	resultStream := simRepository.SearchStream(ctx, searchCh)

	for item := range resultStream.ToChannel() {
		if item.Error() {
			return c.Status(500).JSON(fiber.Map{
				"error": item.E.Error(),
			})
		}
		results = append(results, item.V)
	}

	return c.JSON(fiber.Map{
		"results": results,
		"count":   len(results),
		"query":   query,
	})
}

// ReactiveStatsHandler returns aggregated event statistics
func ReactiveStatsHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get stats for last 5 seconds
	statsStream := eventBroadcaster.AggregateStats(ctx, 5*time.Second)

	// Get latest stats
	select {
	case item := <-statsStream.ToChannel():
		if item.Error() {
			return c.Status(500).JSON(fiber.Map{
				"error": item.E.Error(),
			})
		}
		return c.JSON(item.V)

	case <-time.After(6 * time.Second):
		return c.Status(408).JSON(fiber.Map{
			"error": "Timeout waiting for stats",
		})
	}
}

// EmitSimEvent is a helper to emit SIM-related events
func EmitSimEvent(eventType reactive.EventType, sim interface{}, userID string) {
	eventBroadcaster.Emit(eventType, sim, userID)
}
