package events

import (
	"context"
	"sync"
)

// mockCacheInvalidator is a minimal test double for CacheInvalidator that
// records patterns and signals when it is called. It is safe for concurrent
// use: the handler invalidates asynchronously, so the mock is written from a
// separate goroutine while tests read it.
type mockCacheInvalidator struct {
	mu             sync.Mutex
	calledPatterns []string
	called         chan struct{}
}

func newMockCacheInvalidator() *mockCacheInvalidator {
	return &mockCacheInvalidator{
		calledPatterns: make([]string, 0),
		called:         make(chan struct{}, 10),
	}
}

func (m *mockCacheInvalidator) DeletePattern(_ context.Context, pattern string) error {
	m.mu.Lock()
	m.calledPatterns = append(m.calledPatterns, pattern)
	m.mu.Unlock()
	// Non-blocking signal for tests
	select {
	case m.called <- struct{}{}:
	default:
	}
	return nil
}

// patterns returns a copy of the recorded patterns for race-free inspection.
func (m *mockCacheInvalidator) patterns() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.calledPatterns...)
}

// Handler invalidates cache asynchronously; wait briefly for it to run.
