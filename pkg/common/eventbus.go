package common

import (
	"fmt"
	"sync"
)

// Event represents a system event.
type Event struct {
	Type    string
	Payload interface{}
}

// EventHandler is a callback for events.
type EventHandler func(Event)

// EventBus implements a publish-subscribe event system.
type EventBus struct {
	mu         sync.RWMutex
	subscribers map[string][]EventHandler
	logger     *Logger
}

// NewEventBus creates a new EventBus.
func NewEventBus(logger *Logger) *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
		logger:      logger,
	}
}

// Subscribe registers a handler for an event type.
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
	eb.logger.Debug("subscribed to event: %s", eventType)
}

// Publish sends an event to all subscribers asynchronously.
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	eb.logger.Debug("publishing event: %s (handlers=%d)", event.Type, len(handlers))
	for _, h := range handlers {
		handler := h
		go func() {
			defer func() {
				if r := recover(); r != nil {
					eb.logger.Error("event handler panic for %s: %v", event.Type, r)
				}
			}()
			handler(event)
		}()
	}
}

// PublishSync sends an event synchronously.
func (eb *EventBus) PublishSync(event Event) error {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	for _, h := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					eb.logger.Error("event handler panic for %s: %v", event.Type, r)
				}
			}()
			h(event)
		}()
	}
	return nil
}

// Event type constants.
const (
	EventBlockMined    = "block.mined"
	EventDGUpdated     = "dg.updated"
	EventTaskGenerated = "task.generated"
	EventTaskCompleted = "task.completed"

	// APEX integration events
	EventAPEXCycle     = "apex.cycle"      // APEX evolution cycle completed
	EventAPEXTierUp    = "apex.tier.up"    // APEX tier increased
	EventSignalRelayed = "signal.relayed"  // PHI_APEX signal relayed
)

// String returns a string representation.
func (e Event) String() string {
	return fmt.Sprintf("Event{type=%s}", e.Type)
}
