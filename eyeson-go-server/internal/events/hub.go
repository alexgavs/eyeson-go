// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

// Package events provides a real-time event broadcasting hub for Server-Sent Events (SSE).
package events

import (
	"encoding/json"
	"eyeson-go-server/internal/models"
	"log"
	"sync"
	"time"
)

// Client represents a single connected SSE client.
type Client chan []byte

// Hub manages the set of active clients and broadcasts messages to them.
type Hub struct {
	clients    map[Client]bool
	broadcast  chan []byte
	register   chan Client
	unregister chan Client
	mu         sync.Mutex
}

// Global instance of the Hub
var MainHub = NewHub()

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan Client),
		unregister: make(chan Client),
	}
}

// Run starts the hub's event loop.
func (h *Hub) Run() {
	log.Println("[EventHub] Starting hub...")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			log.Printf("[EventHub] Client registered. Total clients: %d", len(h.clients))
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client)
				log.Printf("[EventHub] Client unregistered. Total clients: %d", len(h.clients))
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client <- message:
				default:
					// If the client's channel is full, assume it's disconnected.
					close(client)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// BroadcastEvent sends an event to all connected clients.
func (h *Hub) BroadcastEvent(eventType string, payload interface{}) {
	event := models.Event{
		Type: eventType,
		Data: payload,
	}
	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("[EventHub] ERROR marshaling event: %v", err)
		return
	}
	
	// Add a small delay to allow the message to be processed,
	// especially if broadcasting in a tight loop.
	time.Sleep(10 * time.Millisecond)
	h.broadcast <- message
}

// RegisterClient adds a new client to the hub.
func (h *Hub) RegisterClient(client Client) {
	h.register <- client
}

// UnregisterClient removes a client from the hub.
func (h *Hub) UnregisterClient(client Client) {
	h.unregister <- client
}
