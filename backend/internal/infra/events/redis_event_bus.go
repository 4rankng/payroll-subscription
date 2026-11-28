package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"api-server/internal/domain"

	"github.com/redis/go-redis/v9"
)

// RedisEventBus is a distributed event bus backed by Redis with worker pool
type RedisEventBus struct {
	redis          *redis.Client
	handlers       map[string][]domain.EventHandler
	globalHandlers []domain.EventHandler
	mu             sync.RWMutex
	logger         *slog.Logger
	eventRegistry  *EventRegistry // Event type registry for deserialization
	metrics        *EventBusMetrics

	// Worker pool configuration
	eventQueue chan *redisEventTask
	workers    int
	queueSize  int
	wg         sync.WaitGroup
	stopCh     chan struct{}

	// Redis configuration
	eventStream    string        // Redis stream name
	consumerGroup  string        // Consumer group for distributed processing
	consumerName   string        // Unique consumer ID
	eventTTL       time.Duration // How long to keep events in Redis
	processingLock time.Duration // Lock timeout for processing events
}

type redisEventTask struct {
	ctx       context.Context
	event     domain.DomainEvent
	handler   domain.EventHandler
	eventType string
	eventID   string // Redis stream message ID
}

// StoredEvent represents an event stored in Redis
type StoredEvent struct {
	EventType   string          `json:"event_type"`
	EventData   json.RawMessage `json:"event_data"`
	AggregateID uint            `json:"aggregate_id"`
	UserID      uint            `json:"user_id"`
	OccurredAt  time.Time       `json:"occurred_at"`
}

// NewRedisEventBus creates a distributed event bus with Redis backing

// Initialize event registry

// Keep events for 24 hours

// Create consumer group (idempotent)

// Load existing metrics from Redis

// Start worker pool

// Start Redis stream consumer

// createConsumerGroup creates the Redis consumer group if it doesn't exist

// Create stream if it doesn't exist by adding a dummy message

// Create consumer group (MKSTREAM creates stream if needed)

// loadMetricsFromRedis loads persisted metrics from Redis on startup

// Load each metric from Redis

// Initialize atomic values with persisted data

// incrementMetric atomically increments both in-memory and Redis metric

// Increment in-memory counter

// Persist to Redis asynchronously (fire-and-forget to avoid blocking)

// Publish publishes events to Redis stream

// Track published event

// Serialize event to JSON

// Store event in Redis stream

// Add to Redis stream with TTL

// Keep last 10K events to prevent unbounded growth
// Allow approximate trimming for performance

// consumeFromRedis continuously reads events from Redis stream

// Read from Redis stream

// ">" means new messages
// Read up to 10 messages
// Block for 1 second

// Process each message

// processRedisMessage processes a single message from Redis with race condition protection

// Deserialize stored event

// RACE CONDITION PROTECTION: Use Redis lock to ensure only one consumer processes this event

// Ensure lock is released

// Dispatch to local handlers

// ACK message in Redis

// acquireLock acquires a distributed lock using Redis SETNX

// Use SET with NX (not exists) and EX (expiry)

// releaseLock releases a distributed lock

// Only delete if we own the lock

// dispatchToHandlers dispatches event to registered handlers

// Reconstruct domain event from stored data

// Collect handlers

// Event-specific handlers

// Global handlers

// Queue tasks for workers

// Task queued successfully

// Queue is full - track dropped event

// deserializeEvent reconstructs a domain event from stored data

// ackMessage acknowledges a message in Redis stream

// worker processes events from the queue

// handleEvent processes a single event task with panic recovery

// Track processing time

// Track panics

// Detach context so event handlers run independently from Redis connection context

// Handle event

// Subscribe registers a handler for a specific event type

// SubscribeAll registers a handler for all events

// Shutdown gracefully stops the event bus

// Close stop channel to signal workers and consumer

// Wait for workers to finish with timeout

// GetMetrics returns current event bus metrics snapshot

// QueueDepth returns current number of events in queue

// AvgProcessingTime returns average event processing time
