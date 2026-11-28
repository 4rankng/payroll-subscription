package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"api-server/internal/pkg/clock"

	"gorm.io/gorm"
)

// BlacklistReason represents the reason for token blacklisting
type BlacklistReason string

const (
	BlacklistReasonLogout      BlacklistReason = "logout"
	BlacklistReasonSecurity    BlacklistReason = "security"
	BlacklistReasonExpired     BlacklistReason = "expired"
	BlacklistReasonAdminRevoke BlacklistReason = "admin_revoke"
)

// AuditAction represents the type of action performed
type AuditAction string

const (
	AuditActionCreate         AuditAction = "CREATE"
	AuditActionUpdate         AuditAction = "UPDATE"
	AuditActionDelete         AuditAction = "DELETE"
	AuditActionApprove        AuditAction = "APPROVE"
	AuditActionReject         AuditAction = "REJECT"
	AuditActionBulkApprove    AuditAction = "BULK_APPROVE"
	AuditActionBulkReject     AuditAction = "BULK_REJECT"
	AuditActionBulkReset      AuditAction = "BULK_RESET"
	AuditActionBulkCreate     AuditAction = "BULK_CREATE"
	AuditActionLogin          AuditAction = "LOGIN"
	AuditActionLogout         AuditAction = "LOGOUT"
	AuditActionChangePassword AuditAction = "CHANGE_PASSWORD"
	AuditActionView           AuditAction = "VIEW"
	AuditActionExport         AuditAction = "EXPORT"
	AuditActionImport         AuditAction = "IMPORT"
)

// EntityType represents the type of entity being audited
type EntityType string

const (
	EntityTypeUser            EntityType = "user"
	EntityTypeProject         EntityType = "project"
	EntityTypeEmployee        EntityType = "employee"
	EntityTypeProjectEmployee EntityType = "project_employee"
	EntityTypeProjectUser     EntityType = "project_user"
	EntityTypeEmployeeUser    EntityType = "employee_user"
	EntityTypePayrate         EntityType = "payrate"
	EntityTypeTimesheet       EntityType = "timesheet"
	EntityTypeSettings        EntityType = "settings"
	EntityTypeAsset           EntityType = "asset"
)

// BlacklistedToken represents a blacklisted JWT token
type BlacklistedToken struct {
	ID            uint            `json:"id" gorm:"primarykey;type:bigint unsigned"`
	TokenJTI      string          `json:"token_jti" gorm:"type:varchar(255);not null;uniqueIndex;comment:'JWT ID'"`
	UserID        uint            `json:"user_id" gorm:"not null;type:bigint unsigned"`
	ExpiresAt     time.Time       `json:"expires_at" gorm:"not null"`
	BlacklistedAt time.Time       `json:"blacklisted_at"`
	Reason        BlacklistReason `json:"reason" gorm:"type:enum('logout','security','expired','admin_revoke');default:'logout'"`

	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         uint        `json:"id"          gorm:"primarykey;type:bigint unsigned"`
	UserID     uint        `json:"user_id"     gorm:"not null;type:bigint unsigned;index"`
	Action     AuditAction `json:"action"      gorm:"type:varchar(50);not null;index:idx_action_entity"`
	EntityType EntityType  `json:"entity_type" gorm:"type:varchar(50);not null;index:idx_action_entity"`
	EntityID   *uint       `json:"entity_id"   gorm:"type:bigint unsigned"`
	Message    string      `json:"message"     gorm:"type:text;not null"`
	IPAddress  *string     `json:"ip_address"  gorm:"type:varchar(45)"`
	Browser    *string     `json:"browser"     gorm:"type:varchar(100)"`
	Platform   *string     `json:"platform"    gorm:"type:varchar(100)"`
	Metadata   *string     `json:"metadata"    gorm:"type:json"`
	CreatedAt  time.Time   `json:"created_at"  gorm:"index"`

	// Joined fields (populated by repository queries, not stored in DB)
	// Using column tag with read-only (->) so GORM populates these from SELECT aliases
	UserFullname string `json:"user_fullname" gorm:"column:user_fullname;->"`
	UserUsername string `json:"user_username" gorm:"column:user_username;->"`
	UserRole     string `json:"user_role"     gorm:"column:user_role;->"`

	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}

// BlacklistedTokenRepository defines the interface for blacklisted token persistence operations
type BlacklistedTokenRepository interface {
	Create(ctx context.Context, token *BlacklistedToken) error
	GetByJTI(ctx context.Context, jti string) (*BlacklistedToken, error)
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
	DeleteExpired(ctx context.Context) error
	List(ctx context.Context, filters BlacklistFilters) ([]*BlacklistedToken, error)
	Count(ctx context.Context, filters BlacklistFilters) (int64, error)
	GetByUser(ctx context.Context, userID uint) ([]*BlacklistedToken, error)
	BlacklistToken(ctx context.Context, jti string, userID uint, expiresAt time.Time, reason BlacklistReason) error
}

// AuditLogRepository defines the interface for audit log persistence operations
type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	GetByID(ctx context.Context, id uint) (*AuditLog, error)
	List(ctx context.Context, filters AuditFilters) ([]*AuditLog, error)
	Count(ctx context.Context, filters AuditFilters) (int64, error)
	DeleteOldLogs(ctx context.Context, olderThan time.Time) error

	// Analytics aggregations for the System Health dashboard. These run
	// GROUP BY + COUNT(DISTINCT) at the database so the application never
	// scans unbounded numbers of raw audit rows.
	GetBrowserPlatformStats(ctx context.Context, since time.Time) ([]BrowserPlatformStat, error)
	GetBrowserPlatformUsers(ctx context.Context, browser, platform string, since time.Time) ([]BrowserPlatformUser, error)
	GetOSGroupedStats(ctx context.Context, since time.Time) ([]OSGroupStat, error)
	GetBrowserGroupedStats(ctx context.Context, since time.Time) ([]BrowserGroupStat, error)
	GetOSGroupedUsers(ctx context.Context, osFamily string, since time.Time) ([]BrowserPlatformUser, error)
	GetBrowserGroupedUsers(ctx context.Context, browserFamily string, since time.Time) ([]BrowserPlatformUser, error)
}

// BlacklistFilters represents filtering options for blacklisted token queries
type BlacklistFilters struct {
	UserID    *uint
	Reason    []BlacklistReason
	FromDate  *time.Time
	ToDate    *time.Time
	Expired   *bool
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// AuditFilters represents filtering options for audit log queries
type AuditFilters struct {
	UserID     *uint
	Action     []AuditAction
	EntityType []EntityType
	EntityID   *uint
	FromDate   *time.Time
	ToDate     *time.Time
	IPAddress  *string
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
}

// BrowserPlatformStat represents unique user/action counts for a browser+platform combination.
type BrowserPlatformStat struct {
	Browser      string `json:"browser"`
	Platform     string `json:"platform"`
	UniqueUsers  int    `json:"unique_users"`
	TotalActions int    `json:"total_actions"`
}

// BrowserPlatformUser represents a user who used a specific browser+platform (or family) combination.
type BrowserPlatformUser struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Role     string `json:"role"`
	Actions  int    `json:"actions"`
	LastSeen string `json:"last_seen"`
}

// VersionStat holds user/action counts for a single version within a family.
type VersionStat struct {
	Version      string `json:"version"`
	UniqueUsers  int    `json:"unique_users"`
	TotalActions int    `json:"total_actions"`
}

// OSGroupStat holds aggregated stats for one OS family with per-version breakdown.
type OSGroupStat struct {
	OSFamily     string        `json:"os_family"`
	UniqueUsers  int           `json:"unique_users"`
	TotalActions int           `json:"total_actions"`
	Versions     []VersionStat `json:"versions"`
}

// BrowserGroupStat holds aggregated stats for one browser family with per-version breakdown.
type BrowserGroupStat struct {
	BrowserFamily string        `json:"browser_family"`
	UniqueUsers   int           `json:"unique_users"`
	TotalActions  int           `json:"total_actions"`
	Versions      []VersionStat `json:"versions"`
}

// GORM hooks for AuditLog
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.Action == "" {
		return NewValidationError("audit log action is required")
	}
	if a.EntityType == "" {
		return NewValidationError("audit log entity_type is required")
	}
	return nil
}

// SetMetadata JSON-marshals v and stores it in the Metadata field.
// If v is nil, Metadata is set to nil.
func (a *AuditLog) SetMetadata(v map[string]interface{}) error {
	if v == nil {
		a.Metadata = nil
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s := string(b)
	a.Metadata = &s
	return nil
}

// GetMetadata JSON-unmarshals the Metadata field and returns the map.
// Returns nil, nil when Metadata is nil.
func (a *AuditLog) GetMetadata() (map[string]interface{}, error) {
	if a.Metadata == nil {
		return nil, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(*a.Metadata), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// GORM hooks for BlacklistedToken
func (bt *BlacklistedToken) BeforeCreate(tx *gorm.DB) error {
	if bt.BlacklistedAt.IsZero() {
		bt.BlacklistedAt = clock.Now()
	}
	return nil
}

// BlacklistedToken methods

// ValidateTokenJTI validates the token JTI
func (bt *BlacklistedToken) ValidateTokenJTI() error {
	if bt.TokenJTI == "" {
		return NewValidationError("token JTI is required")
	}
	return nil
}

// ValidateUserID validates the user ID
func (bt *BlacklistedToken) ValidateUserID() error {
	if bt.UserID == 0 {
		return NewValidationError("user ID is required")
	}
	return nil
}

// ValidateExpiresAt validates the expiration time
func (bt *BlacklistedToken) ValidateExpiresAt() error {
	if bt.ExpiresAt.IsZero() {
		return NewValidationError("expiration time is required")
	}
	return nil
}

// ValidateReason validates the blacklist reason
func (bt *BlacklistedToken) ValidateReason() error {
	validReasons := map[BlacklistReason]bool{
		BlacklistReasonLogout:      true,
		BlacklistReasonSecurity:    true,
		BlacklistReasonExpired:     true,
		BlacklistReasonAdminRevoke: true,
	}

	if !validReasons[bt.Reason] {
		return NewValidationError("invalid blacklist reason")
	}

	return nil
}

// IsValid validates the entire blacklisted token
func (bt *BlacklistedToken) IsValid() error {
	if err := bt.ValidateTokenJTI(); err != nil {
		return err
	}
	if err := bt.ValidateUserID(); err != nil {
		return err
	}
	if err := bt.ValidateExpiresAt(); err != nil {
		return err
	}
	if err := bt.ValidateReason(); err != nil {
		return err
	}
	return nil
}

// IsExpired returns true if the token has expired
func (bt *BlacklistedToken) IsExpired() bool {
	return clock.Now().After(bt.ExpiresAt)
}

// CanBeDeleted returns true if the token can be deleted (expired)
func (bt *BlacklistedToken) CanBeDeleted() bool {
	return bt.IsExpired()
}

// Helper functions

// Deprecated: Use BuildEventAuditMessage from audit_templates.go for new code.
// This function remains for backward compatibility with financial event factories.
// If entityName is provided, it will be included in the message (e.g., "đã tạo người dùng mới Tài Văn Lộc").

// Append entity name if provided, except for self-referential actions
// (LOGIN, LOGOUT, CHANGE_PASSWORD don't need entity names as they are performed by the user on themselves)

// Deprecated: Use BuildEventAuditMessage from audit_templates.go for new code.
// This function remains for backward compatibility with bulk timesheet events.
// Example: "tạo 18 chấm công", "duyệt 25 chấm công"
func BuildBulkAuditMessage(action AuditAction, entityType EntityType, count int) string {
	var verb string
	switch action {
	case AuditActionBulkCreate:
		verb = "tạo"
	case AuditActionBulkApprove:
		verb = "duyệt"
	case AuditActionBulkReject:
		verb = "từ chối"
	case AuditActionBulkReset:
		verb = "reset"
	default:
		verb = "xử lý"
	}

	var entityName string
	switch entityType {
	case EntityTypeTimesheet:
		entityName = "chấm công"
	case EntityTypeUser:
		entityName = "người dùng"
	case EntityTypeEmployee:
		entityName = "nhân viên"
	case EntityTypeProject:
		entityName = "dự án"
	case EntityTypeProjectEmployee:
		entityName = "phân công nhân viên"
	case EntityTypeProjectUser:
		entityName = "chia sẻ dự án"
	case EntityTypeEmployeeUser:
		entityName = "chia sẻ nhân viên"
	case EntityTypePayrate:
		entityName = "mức lương"
	case EntityTypeSettings:
		entityName = "cài đặt"
	case EntityTypeAsset:
		entityName = "tệp tin"
	default:
		entityName = "bản ghi"
	}

	return fmt.Sprintf("%s %d %s", verb, count, entityName)
}

// CreateBlacklistedToken creates a new blacklisted token entry
