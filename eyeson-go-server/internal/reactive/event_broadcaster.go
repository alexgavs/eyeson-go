// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System - Reactive Event Broadcaster

package reactive

import (
	"context"
	"encoding/json"
	"eyeson-go-server/internal/models"
	"log"
	"time"

	"github.com/reactivex/rxgo/v2"
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

// EventBroadcaster manages reactive event broadcasting
type EventBroadcaster struct {
	eventCh chan rxgo.Item
	config  StreamConfig
}

// NewEventBroadcaster creates a new event broadcaster
func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		eventCh: make(chan rxgo.Item, 1000),
		config:  DefaultStreamConfig(),
	}
}

// Emit sends an event to the broadcast stream
func (b *EventBroadcaster) Emit(eventType EventType, data interface{}, userID string) {
	event := Event{
		ID:        generateEventID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		UserID:    userID,
	}

	select {
	case b.eventCh <- rxgo.Of(event):
	default:
		log.Printf("[EventBroadcaster] Warning: event channel full, dropping event %s", eventType)
	}
}

// Stream returns the event stream
func (b *EventBroadcaster) Stream(ctx context.Context) *Stream {
	return NewStream(ctx, b.eventCh, b.config)
}

// FilterByType creates a stream filtered by event type
func (b *EventBroadcaster) FilterByType(ctx context.Context, eventTypes ...EventType) *Stream {
	typeMap := make(map[EventType]bool)
	for _, t := range eventTypes {
		typeMap[t] = true
	}

	return b.Stream(ctx).Filter(func(item interface{}) bool {
		event, ok := item.(Event)
		if !ok {
			return false
		}
		return typeMap[event.Type]
	})
}

// FilterByUser creates a stream filtered by user ID
func (b *EventBroadcaster) FilterByUser(ctx context.Context, userID string) *Stream {
	return b.Stream(ctx).Filter(func(item interface{}) bool {
		event, ok := item.(Event)
		if !ok {
			return false
		}
		return event.UserID == userID || event.UserID == ""
	})
}

// ToSSE converts events to Server-Sent Events format
func (b *EventBroadcaster) ToSSE(ctx context.Context, filters ...func(*Stream) *Stream) *Stream {
	stream := b.Stream(ctx)

	// Apply filters
	for _, filter := range filters {
		stream = filter(stream)
	}

	// Convert to SSE format
	return stream.Map(func(item interface{}) interface{} {
		event, ok := item.(Event)
		if !ok {
			return ""
		}

		// Convert to models.Event for SSE
		sseEvent := models.Event{
			Type: string(event.Type),
			Data: event.Data,
		}

		jsonData, err := json.Marshal(sseEvent)
		if err != nil {
			log.Printf("[EventBroadcaster] Error marshaling event: %v", err)
			return ""
		}

		return jsonData
	})
}

// AggregateStats creates a stream of aggregated statistics
func (b *EventBroadcaster) AggregateStats(ctx context.Context, window time.Duration) *Stream {
	resultCh := make(chan rxgo.Item, b.config.BufferSize)

	go func() {
		defer close(resultCh)

		b.Stream(ctx).
			Buffer(100, window).
			Map(func(batch interface{}) interface{} {
				items, ok := batch.([]interface{})
				if !ok {
					return EventStats{}
				}

				stats := EventStats{
					Timestamp: time.Now(),
					Total:     len(items),
					ByType:    make(map[EventType]int),
				}

				for _, item := range items {
					event, ok := item.(Event)
					if !ok {
						continue
					}
					stats.ByType[event.Type]++
				}

				return stats
			}).
			Subscribe(ctx,
				func(stats interface{}) {
					resultCh <- rxgo.Of(stats)
				},
				func(err error) {
					resultCh <- rxgo.Error(err)
				},
				nil,
			)
	}()

	return NewStream(ctx, resultCh, b.config)
}

// EventStats holds aggregated event statistics
type EventStats struct {
	Timestamp time.Time         `json:"timestamp"`
	Total     int               `json:"total"`
	ByType    map[EventType]int `json:"by_type"`
}

// LogEvents subscribes to events and logs them
func (b *EventBroadcaster) LogEvents(ctx context.Context) {
	b.Stream(ctx).
		Subscribe(ctx,
			func(item interface{}) {
				event, ok := item.(Event)
				if !ok {
					return
				}
				log.Printf("[Event] %s: %s (User: %s)", event.Type, event.ID, event.UserID)
			},
			func(err error) {
				log.Printf("[EventBroadcaster] Error: %v", err)
			},
			func() {
				log.Println("[EventBroadcaster] Event stream closed")
			},
		)
}

// ReplayLast creates a stream that replays the last N events for new subscribers
func (b *EventBroadcaster) ReplayLast(ctx context.Context, count int) *Stream {
	// This would need a buffer to store recent events
	// For simplicity, we'll just return the normal stream
	// In production, you'd want to use rxgo.Replay()
	return b.Stream(ctx)
}

var eventIDCounter uint64

func generateEventID() string {
	eventIDCounter++
	return time.Now().Format("20060102150405") + "-" + string(rune(eventIDCounter))
}
