package events

import (
	"context"
	"encoding/json"
	"testing"

	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
)

func TestAuditEventHandler_PreservesMutationMetadata(t *testing.T) {
	ctx := auditctx.WithFullName(auditctx.WithUserID(context.Background(), 42), "Quản trị viên")

	tests := []struct {
		name     string
		event    domain.DomainEvent
		expected map[string]interface{}
	}{
		{
			name:  "data import",
			event: domain.NewDataImportedEvent(ctx, "employee", 12, "employees.xlsx"),
			expected: map[string]interface{}{
				"data_type":    "employee",
				"record_count": float64(12),
				"file_name":    "employees.xlsx",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enqueuer := &mockAuditEnqueuer{}
			handler := NewAuditEventHandler(enqueuer)

			if err := handler.Handle(ctx, tt.event); err != nil {
				t.Fatalf("Handle: %v", err)
			}
			if len(enqueuer.payloads) != 1 {
				t.Fatalf("expected one audit payload, got %d", len(enqueuer.payloads))
			}

			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(enqueuer.payloads[0].MetadataJSON), &metadata); err != nil {
				t.Fatalf("metadata unmarshal: %v", err)
			}
			for key, expected := range tt.expected {
				if actual, ok := metadata[key]; !ok || actual != expected {
					t.Errorf("metadata[%q] = %#v, want %#v", key, actual, expected)
				}
			}
		})
	}
}
