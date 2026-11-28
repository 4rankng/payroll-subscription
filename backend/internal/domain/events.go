package domain

import (
	"context"
	"time"
)

// DomainEvent represents a domain event that occurred in the system
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
	AggregateID() uint
	UserID() uint
	GetAuditMessage() string

	// New structured audit fields
	GetAction() AuditAction
	GetEntityType() EntityType
	GetIPAddress() string
	GetUserAgent() string
}

// BaseEvent provides common fields for all domain events
type BaseEvent struct {
	EventName    string
	Timestamp    time.Time
	EntityID     uint
	ActorUserID  uint
	AuditMessage string

	// New structured audit fields
	Action     AuditAction
	EntityType EntityType
	IPAddress  string
	UserAgent  string
}

func (e BaseEvent) EventType() string {
	return e.EventName
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

func (e BaseEvent) AggregateID() uint {
	return e.EntityID
}

func (e BaseEvent) UserID() uint {
	return e.ActorUserID
}

func (e BaseEvent) GetAuditMessage() string {
	return e.AuditMessage
}

func (e BaseEvent) GetAction() AuditAction {
	return e.Action
}

func (e BaseEvent) GetEntityType() EntityType {
	return e.EntityType
}

func (e BaseEvent) GetIPAddress() string {
	return e.IPAddress
}

func (e BaseEvent) GetUserAgent() string {
	return e.UserAgent
}

// User Events

type UserCreatedEvent struct {
	BaseEvent
	Username string
	Fullname string
	Email    *string
	Role     UserRole
}

type UserUpdatedEvent struct {
	BaseEvent
	Username      string
	Fullname      string
	Email         *string
	Role          UserRole
	ChangedFields map[string]FieldChange
}

type UserDeletedEvent struct {
	BaseEvent
	Username string
	Fullname string
}

type PasswordChangedEvent struct {
	BaseEvent
	Method string // "self_service" or "admin_reset"
}

type UserLoginEvent struct {
	BaseEvent
	Username            string
	IPAddress           string
	UserAgent           string
	Success             bool
	Reason              string            // Only set for failed login
	AttemptedIdentifier string            // Value user typed in the username box
	Location            map[string]string // IP geolocation (country, city, region)
}

type UserLogoutEvent struct {
	BaseEvent
	Username string
}

// Employee Events

type EmployeeCreatedEvent struct {
	BaseEvent
	Fullname string
	CCCD     string
}

type EmployeeUpdatedEvent struct {
	BaseEvent
	Fullname      string
	CCCD          string
	ChangedFields map[string]FieldChange
}

type EmployeeDeletedEvent struct {
	BaseEvent
	Fullname string
	CCCD     string
}

type EmployeeProjectAssignmentsRemovedEvent struct {
	BaseEvent
	Fullname string
	CCCD     string
}

type EmployeeProfileUpdatedEvent struct {
	BaseEvent
	Fullname      string
	ChangedFields map[string]FieldChange
}

type EmployeeNameUpdatedEvent struct {
	BaseEvent
	OldFullName string
	NewFullName string
	CCCD        string
}

type ProjectEmployeesSyncedEvent struct {
	BaseEvent
	EmployeeID   uint
	OldFullName  string
	NewFullName  string
	UpdatedCount int
	SyncedAt     time.Time
}

// Project Events

type ProjectCreatedEvent struct {
	BaseEvent
	Name       string
	Code       string
	ClientName string
}

type ProjectUpdatedEvent struct {
	BaseEvent
	Name          string
	Code          string
	ClientName    string
	ChangedFields map[string]FieldChange
}

type ProjectDeletedEvent struct {
	BaseEvent
	Name string
	Code string
}

// ProjectEmployee Events

type ProjectEmployeeUpdatedEvent struct {
	BaseEvent
	ProjectID    uint
	ProjectName  string
	EmployeeID   uint
	EmployeeName string
	Status       string
}

// Timesheet Events

type TimesheetCreatedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
	Date         time.Time
}

type TimesheetUpdatedEvent struct {
	BaseEvent
	EmployeeID    uint
	EmployeeName  string
	ProjectID     uint
	ProjectName   string
	ChangedFields map[string]FieldChange
}

type TimesheetDeletedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
}

type TimesheetApprovedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
	ApproverID   uint
}

type TimesheetRejectedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
	ApproverID   uint
}

type TimesheetBulkApprovedEvent struct {
	BaseEvent
	Count      int
	ProjectID  *uint
	ApproverID uint
}

type TimesheetBulkRejectedEvent struct {
	BaseEvent
	Count      int
	ProjectID  *uint
	ApproverID uint
}

type TimesheetBulkResetEvent struct {
	BaseEvent
	Count     int
	ProjectID *uint
	ResetBy   uint
}

type TimesheetBulkCreatedEvent struct {
	BaseEvent
	Count     int
	ProjectID *uint
	CreatedBy uint
}

// Payrate Events

type PayrateCreatedEvent struct {
	BaseEvent
	ProjectID   uint
	ProjectName string
	EmployeeID  uint
	Rate        int64
}

type PayrateUpdatedEvent struct {
	BaseEvent
	ProjectID   uint
	ProjectName string
	EmployeeID  uint
	Rate        int64
}

type PayrateDeletedEvent struct {
	BaseEvent
	ProjectID   uint
	ProjectName string
	EmployeeID  uint
}

// Asset Events

type AssetCreatedEvent struct {
	BaseEvent
	FileName string
}

type AssetDeletedEvent struct {
	BaseEvent
	FileName string
}

// Settings Events

type SettingsCreatedEvent struct {
	BaseEvent
	Key string
}

type SettingsUpdatedEvent struct {
	BaseEvent
	Key string
}

type SettingsDeletedEvent struct {
	BaseEvent
	Key string
}

// Import/Export Events

type DataImportedEvent struct {
	BaseEvent
	DataType    string
	RecordCount int
	FileName    string
}

type DataExportedEvent struct {
	BaseEvent
	DataType    string
	RecordCount int
	FileName    string
}

// ProjectEmployee Events

type ProjectEmployeeCreatedEvent struct {
	BaseEvent
	ProjectID    uint
	ProjectName  string
	EmployeeID   uint
	EmployeeName string
	Role         string
}

type ProjectEmployeeDeletedEvent struct {
	BaseEvent
	ProjectID    uint
	ProjectName  string
	EmployeeID   uint
	EmployeeName string
	Reason       string
}

// Permission Events

type EmployeeAccessGrantedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
	Role         string
	GrantedBy    uint
}

type EmployeeAccessRevokedEvent struct {
	BaseEvent
	EmployeeID    uint
	EmployeeName  string
	ProjectID     uint
	ProjectName   string
	RevokedBy     uint
	RevokedUserID uint
}

type ProjectAccessGrantedEvent struct {
	BaseEvent
	ProjectID    uint
	ProjectName  string
	EmployeeID   uint
	EmployeeName string
	Role         string
	GrantedBy    uint
}

type ProjectAccessRevokedEvent struct {
	BaseEvent
	ProjectID    uint
	ProjectName  string
	EmployeeID   uint
	EmployeeName string
	RevokedBy    uint
}

// EventBus defines the interface for publishing and subscribing to domain events
type EventBus interface {
	Publish(ctx context.Context, events ...DomainEvent) error
	Subscribe(eventType string, handler EventHandler)
	SubscribeAll(handler EventHandler)
}

// EventHandler handles domain events
type EventHandler interface {
	Handle(ctx context.Context, event DomainEvent) error
	CanHandle(eventType string) bool
}
