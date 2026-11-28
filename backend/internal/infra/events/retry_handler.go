package events

import (
	"log/slog"
	"time"

	"api-server/internal/domain"
)

// RetryHandler wraps a domain.EventHandler with configurable retry and exponential backoff.
type RetryHandler struct {
	inner     domain.EventHandler
	maxTries  int
	baseDelay time.Duration
	logger    *slog.Logger
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxTries  int           // total attempts (including initial)
	BaseDelay time.Duration // initial backoff delay
}

// DefaultRetryConfig returns sensible defaults: 3 attempts, 100ms base delay.

// NewRetryHandler wraps a handler with retry logic.

// Handle executes the inner handler with retry and exponential backoff.

// CanHandle delegates to the inner handler.
