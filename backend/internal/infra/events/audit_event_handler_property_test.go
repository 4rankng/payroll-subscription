// Feature: audit-log-revamp, Property 5: AuditEventHandler persists structured fields from event
// Feature: audit-log-revamp, Property 7: Failed login metadata is stored
// Feature: audit-log-revamp, Property 13: BulkTransfer metadata completeness
package events

import (
	"time"

	"api-server/internal/domain"
	asynqinfra "api-server/internal/infra/asynq"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
)

// mockAuditEnqueuer captures payloads enqueued via EnqueueAuditLogWrite.
type mockAuditEnqueuer struct {
	payloads []asynqinfra.AuditLogWritePayload
}

func (m *mockAuditEnqueuer) EnqueueAuditLogWrite(p asynqinfra.AuditLogWritePayload) error {
	m.payloads = append(m.payloads, p)
	return nil
}

// mockDomainEvent is a minimal DomainEvent implementation for Property 5.
type mockDomainEvent struct {
	action     domain.AuditAction
	entityType domain.EntityType
	entityID   uint
	userID     uint
	ipAddress  string
	userAgent  string
	message    string
}

func (e mockDomainEvent) EventType() string                { return "MockEvent" }
func (e mockDomainEvent) OccurredAt() time.Time            { return time.Now() }
func (e mockDomainEvent) AggregateID() uint                { return e.entityID }
func (e mockDomainEvent) UserID() uint                     { return e.userID }
func (e mockDomainEvent) GetAuditMessage() string          { return e.message }
func (e mockDomainEvent) GetAction() domain.AuditAction    { return e.action }
func (e mockDomainEvent) GetEntityType() domain.EntityType { return e.entityType }
func (e mockDomainEvent) GetIPAddress() string             { return e.ipAddress }
func (e mockDomainEvent) GetUserAgent() string             { return e.userAgent }

// validAuditActionsForHandler lists all valid AuditAction constants.
var validAuditActionsForHandler = []interface{}{
	domain.AuditActionCreate,
	domain.AuditActionUpdate,
	domain.AuditActionDelete,
	domain.AuditActionApprove,
	domain.AuditActionReject,
	domain.AuditActionBulkApprove,
	domain.AuditActionBulkReject,
	domain.AuditActionBulkReset,
	domain.AuditActionBulkCreate,
	domain.AuditActionLogin,
	domain.AuditActionLogout,
	domain.AuditActionChangePassword,
	domain.AuditActionView,
	domain.AuditActionExport,
	domain.AuditActionImport,
}

// validEntityTypesForHandler lists all valid EntityType constants.
var validEntityTypesForHandler = []interface{}{
	domain.EntityTypeUser,
	domain.EntityTypeProject,
	domain.EntityTypeEmployee,
	domain.EntityTypeProjectEmployee,
	domain.EntityTypeProjectUser,
	domain.EntityTypeEmployeeUser,
	domain.EntityTypePayrate,
	domain.EntityTypeTimesheet,
	domain.EntityTypeSettings,
	domain.EntityTypeAsset,
}

func genHandlerAuditAction() gopter.Gen {
	return gen.OneConstOf(validAuditActionsForHandler...)
}

func genHandlerEntityType() gopter.Gen {
	return gen.OneConstOf(validEntityTypesForHandler...)
}

// genNonEmptyAlphaString generates a non-empty alpha string.
func genNonEmptyAlphaString() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 })
}

// genPositiveUint generates a uint > 0.
func genPositiveUint() gopter.Gen {
	return gen.UInt().Map(func(v uint) uint {
		if v == 0 {
			return 1
		}
		return v
	})
}

// TestProperty5_HandlerPersistsStructuredFields verifies that AuditEventHandler
// enqueues a payload whose fields exactly match the values from the domain event.
// UA parsing (Browser/Platform) is deferred to the worker and not tested here.
//
// Validates: Requirements 3.1–3.12, 4.3, 8.4

// entityID
// userID
// ipAddress
// message

// TestProperty7_FailedLoginMetadataIsStored verifies that when a UserLoginEvent
// with Success=false is handled, the enqueued payload metadata contains
// {"success": false, "reason": "<reason>"}.
//
// Validates: Requirements 4.4, 9.2

// TestHandler_DropsEmitsWithUserIDZero_ExceptLogin verifies the filter rule:
// audit rows with user_id=0 are silently dropped except for login attempts
// (failed logins legitimately carry user_id=0). Cron jobs / system writes
// without an actor must not pollute the audit log.

// Non-login event with user_id=0 → dropped.

// Failed login with user_id=0 → kept.

// Non-zero user_id → kept regardless of action.
