package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIncrementTenantThrottle(t *testing.T) {
	tenant := "test-tenant-1"

	// Should not panic
	IncrementTenantThrottle(tenant)
	IncrementTenantThrottle(tenant)
	IncrementTenantThrottle(tenant)

	// Verify metric exists by checking metrics endpoint
	handler := MetricsHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	responseBody := w.Body.String()

	if !strings.Contains(responseBody, "tenant_throttle_total") {
		t.Error("Expected metrics to contain tenant_throttle_total")
	}
}

// Should not panic

// Verify metric exists by checking metrics endpoint

func TestMetricsHandler(t *testing.T) {
	handler := MetricsHandler()

	if handler == nil {
		t.Fatal("MetricsHandler() returned nil")
	}

	// Test that handler responds to requests
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("MetricsHandler response code = %v, want %v", w.Code, http.StatusOK)
	}

	responseBody := w.Body.String()

	// Verify it returns Prometheus metrics
	if !strings.Contains(responseBody, "# HELP") {
		t.Error("Expected metrics response to contain Prometheus format")
	}
}

// Verify all tenants are tracked
