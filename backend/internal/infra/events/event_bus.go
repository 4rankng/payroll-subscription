package events

import (
	"log/slog"
	"sync"

	"api-server/internal/domain"
)

// InMemoryEventBus is a simple in-memory event bus implementation
type InMemoryEventBus struct {
	handlers       map[string][]domain.EventHandler
	globalHandlers []domain.EventHandler
	mu             sync.RWMutex
	logger         *slog.Logger
}

// NewInMemoryEventBus creates a new in-memory event bus

// Publish publishes events to all registered handlers asynchronously

// Process event-specific handlers asynchronously

// Process global handlers asynchronously

// handleEventAsync processes an event handler in a goroutine with panic recovery

// Subscribe registers a handler for a specific event type

// SubscribeAll registers a handler for all events
