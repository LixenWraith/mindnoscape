// Package event handles triggering of operations without direct dependency
package event

import (
	"fmt"
	"sync"
)

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

type Event struct {
	Type EventType
	Data interface{}
}

type EventHandler func(Event)

type EventManager struct {
	subscribers map[EventType][]EventHandler
	mu          sync.RWMutex
}

func NewEventManager() *EventManager {
	return &EventManager{
		subscribers: make(map[EventType][]EventHandler),
	}
}

func (em *EventManager) Subscribe(eventType EventType, handler EventHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.subscribers[eventType] = append(em.subscribers[eventType], handler)
}

func (em *EventManager) Publish(event Event) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	for _, handler := range em.subscribers[event.Type] {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil { // Avoid nil panics
					fmt.Printf("Panic in event handler: %v\n", r)
					// Optionally log the stack trace here
				}
			}()
			h(event)
		}(handler)
	}
}
