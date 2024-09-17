// Package event handles triggering of operations without direct dependency
package event

import (
	"context"
	"sync"

	"mindnoscape/local-app/src/pkg/log"
)

// EventType represents the type of event
type EventType int

const (
	UserDeleted EventType = iota
	MindmapAdded
	MindmapDeleted
	MindmapUpdated
	NodeUpdated
	NodeDeleted
	NodeSorted
	RootNodeRenamed
	MindmapSelected
)

// Event represents an event with its type and associated data
type Event struct {
	Type EventType
	Data interface{}
}

// EventHandler is a function type for event handlers
type EventHandler func(Event)

// EventManager manages event subscriptions and publications
type EventManager struct {
	subscribers map[EventType][]EventHandler
	mu          sync.RWMutex
	logger      *log.Logger
}

// NewEventManager creates a new EventManager instance
func NewEventManager(logger *log.Logger) *EventManager {
	return &EventManager{
		subscribers: make(map[EventType][]EventHandler),
		logger:      logger,
	}
}

// Subscribe adds a new event handler for a specific event type
func (em *EventManager) Subscribe(eventType EventType, handler EventHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.subscribers[eventType] = append(em.subscribers[eventType], handler)
}

// Publish sends an event to all subscribed handlers
func (em *EventManager) Publish(event Event) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	for _, handler := range em.subscribers[event.Type] {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					em.logger.Error(context.Background(), "Panic in event handler", log.Fields{
						"event": event,
						"panic": r,
					})
				}
			}()
			h(event)
		}(handler)
	}
}
