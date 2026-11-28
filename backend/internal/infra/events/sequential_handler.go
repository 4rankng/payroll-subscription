package events

import (
	"api-server/internal/domain"
)

// SequentialEventHandler executes a list of handlers sequentially for each event.
// It is useful when we want to avoid concurrent handler execution for a single event type
// but still keep each handler's own behavior (including coordination and tracking).
type SequentialEventHandler struct {
	handlers []domain.EventHandler
}

// NewSequentialEventHandler creates a new sequential event handler wrapper.

// Handle runs all inner handlers sequentially. If any handler returns an error,
// it stops and returns that error.

// CanHandle returns true if any inner handler can handle the given eventType.
