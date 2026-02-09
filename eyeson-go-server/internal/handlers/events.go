// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"bufio"
	"encoding/json"
	"eyeson-go-server/internal/events"
	"eyeson-go-server/internal/models"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

// EventsHandler establishes an SSE connection with a client.
func EventsHandler(c *fiber.Ctx) error {
	// Set headers for SSE
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// fasthttp way to stream
	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		log.Println("[SSE] Client connected")
		defer log.Println("[SSE] Client disconnected")

		// Create a channel for this client
		clientChan := make(events.Client, 10) // Buffer size 10
		events.MainHub.RegisterClient(clientChan)
		defer events.MainHub.UnregisterClient(clientChan)

		// Send a welcome message
		welcomeEvent := models.Event{
			Type: "CONNECTION_ESTABLISHED",
			Data: fiber.Map{
				"message": "SSE connection successful",
				"time":    time.Now(),
			},
		}

		// Manually format the welcome message
		if jsonData, err := json.Marshal(welcomeEvent); err == nil {
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			w.Flush()
		}

		// Keep-alive ticker
		keepAliveTicker := time.NewTicker(15 * time.Second)
		defer keepAliveTicker.Stop()

		for {
			select {
			case message, ok := <-clientChan:
				if !ok {
					// Channel closed
					return
				}
				// Format message according to SSE spec
				fmt.Fprintf(w, "data: %s\n\n", message)
				err := w.Flush()
				if err != nil {
					log.Printf("[SSE] Error flushing to client, likely disconnected: %v", err)
					return
				}
			case <-keepAliveTicker.C:
				// Send a keep-alive comment to prevent timeouts
				fmt.Fprintf(w, ": keep-alive\n\n")
				err := w.Flush()
				if err != nil {
					log.Printf("[SSE] Error sending keep-alive, likely disconnected: %v", err)
					return
				}
			case <-c.Context().Done():
				// Client has disconnected
				return
			}
		}
	}))

	return nil
}
