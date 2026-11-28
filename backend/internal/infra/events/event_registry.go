package events

import (
	"fmt"
	"sort"

	"api-server/internal/domain"
)

// EventRegistry maps event type names to deserialization functions
type EventRegistry struct {
	deserializers map[string]func([]byte) (domain.DomainEvent, error)
}

// NewEventRegistry creates a new event registry with all known event types

// Register all event types

// Deserialize deserializes event data based on event type
func (r *EventRegistry) Deserialize(eventType string, data []byte) (domain.DomainEvent, error) {
	deserializer, exists := r.deserializers[eventType]
	if !exists {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	return deserializer(data)
}

// registerAllEvents registers all domain event types

// User events

// Employee events

// Project events

// ProjectEmployee events

// Timesheet events

// Payrate events

// Asset events

// Settings events

// Import/Export events

// register adds a deserializer for an event type

// EventTypes returns a sorted list of registered event type names.
func (r *EventRegistry) EventTypes() []string {
	eventTypes := make([]string, 0, len(r.deserializers))
	for eventType := range r.deserializers {
		eventTypes = append(eventTypes, eventType)
	}
	sort.Strings(eventTypes)
	return eventTypes
}

// HasEventType checks if an event type is registered
func (r *EventRegistry) HasEventType(eventType string) bool {
	_, exists := r.deserializers[eventType]
	return exists
}
