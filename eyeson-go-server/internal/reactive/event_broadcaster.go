// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System - Reactive Event Broadcaster

package reactive

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// EventType represents different types of events
type EventType string

const (
	EventSimCreated     EventType = "SIM_CREATED"
	EventSimUpdated     EventType = "SIM_UPDATED"
	EventSimDeleted     EventType = "SIM_DELETED"
	EventSyncStarted    EventType = "SYNC_STARTED"
	EventSyncCompleted  EventType = "SYNC_COMPLETED"
	EventSyncFailed     EventType = "SYNC_FAILED"
	EventTaskQueued     EventType = "TASK_QUEUED"
	EventTaskProcessing EventType = "TASK_PROCESSING"
	EventTaskCompleted  EventType = "TASK_COMPLETED"
	EventTaskFailed     EventType = "TASK_FAILED"
)

// Event represents a system event
type Event struct {
	ID        string      `json:"id"`
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	UserID    string      `json:"user_id,omitempty"`
}

// EventStats holds aggregated event statistics
type EventStats struct {
	Timestamp   time.Time           `json:"timestamp"`
	Total       int64               `json:"total"`
	ByType      map[EventType]int64 `json:"by_type"`
	Subscribers int                 `json:"subscribers"`
}

// EventBroadcaster manages fan-out event broadcasting to multiple SSE subscribers.
// Each subscriber gets its own channel so events are delivered to ALL clients.
type EventBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}

	// Stats tracking (lock-free)
	totalEvents int64
	statsByType sync.Map // EventType -> *int64
}

// NewEventBroadcaster creates a new event broadcaster
func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		subscribers: make(map[chan Event]struct{}),
	}
}

// Subscribe registers a new SSE client and returns its dedicated event channel.
func (b *EventBroadcaster) Subscribe() chan Event {
	ch := make(chan Event, 256)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	count := len(b.subscribers)
	b.mu.Unlock()
	log.Printf("[EventBroadcaster] Subscriber added (total: %d)", count)
	return ch
}

// Unsubscribe removes an SSE client channel.
func (b *EventBroadcaster) Unsubscribe(ch chan Event) {
	b.mu.Lock()
	delete(b.subscribers, ch)
	count := len(b.subscribers)
	b.mu.Unlock()
	close(ch)
	log.Printf("[EventBroadcaster] Subscriber removed (total: %d)", count)
}

// Emit sends an event to ALL subscribed clients (fan-out).
func (b *EventBroadcaster) Emit(eventType EventType, data interface{}, userID string) {
	event := Event{
		ID:        generateEventID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		UserID:    userID,
	}

	// Update stats
	atomic.AddInt64(&b.totalEvents, 1)
	if ptr, ok := b.statsByType.Load(eventType); ok {
		atomic.AddInt64(ptr.(*int64), 1)
	} else {
		var n int64 = 1
		b.statsByType.Store(eventType, &n)
	}

	log.Printf("[Event] %s: %s (User: %s)", event.Type, event.ID, event.UserID)

	// Fan-out to all subscribers
	b.mu.RLock()
	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			log.Printf("[EventBroadcaster] Warning: subscriber channel full, dropping event %s", eventType)
		}
	}
	b.mu.RUnlock()
}

// GetStats returns current aggregated event statistics.
func (b *EventBroadcaster) GetStats() EventStats {
	byType := make(map[EventType]int64)
	b.statsByType.Range(func(key, value interface{}) bool {
		byType[key.(EventType)] = atomic.LoadInt64(value.(*int64))
		return true
	})

	b.mu.RLock()
	subs := len(b.subscribers)
	b.mu.RUnlock()

	return EventStats{
		Timestamp:   time.Now(),
		Total:       atomic.LoadInt64(&b.totalEvents),
		ByType:      byType,
		Subscribers: subs,
	}
}

var eventIDCounter uint64

func generateEventID() string {
	id := atomic.AddUint64(&eventIDCounter, 1)
	return fmt.Sprintf("%s-%d", time.Now().Format("20060102150405"), id)
}
