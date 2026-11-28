package config

import (
	"context"
	"strings"
	"sync"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
)

// Kept business settings keys. The donor percentage/workbook/self-check-in/
// wallet-alert/transfer-bank keys died with the money-movement model
// (PRD v11 §6.4); the subscription fee is the only pricing setting left.
const (
	// Subscription fee (FR-4, PAYSUB-5): flat VND charged per distinct
	// employee per Kỳ (PRD BR-1, default 10,000). Admin-configurable via the
	// generic settings API; the value is snapshotted into each billing
	// report at generation time so later changes never rewrite history.
	SettingKeySubscriptionFeePerEmployeePerKyVnd = "subscription_fee_per_employee_per_ky_vnd"

	// Sao kê statement mailing list (PRD §5.8): JSON arrays of
	// email addresses. Owned by the billing sao kê endpoints, not the
	// generic settings API — ketoan/giamdoc must edit them without holding
	// settings permissions. Missing rows read as empty lists.
	SettingKeySaoKeEmailRecipients = "saoke_email_recipients"
	SettingKeySaoKeEmailCC         = "saoke_email_cc"
)

// DefaultSubscriptionFeePerEmployeePerKy is the flat per-employee-per-Kỳ fee
// used when the settings row is missing or unparsable (PRD BR-1).
const DefaultSubscriptionFeePerEmployeePerKy = 10000

// cacheEntry stores cached value with expiration time
type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

type SettingReader interface {
	GetSettingByKey(ctx context.Context, key string) (*domain.Settings, error)
	GetSettingByKeyAuthoritative(ctx context.Context, key string) (*domain.Settings, error)
	GetSettingByKeyAuthoritativeForUpdate(ctx context.Context, key string) (*domain.Settings, error)
}

// SettingsConfigService provides cached reads of the kept business settings.
type SettingsConfigService struct {
	settingsService SettingReader
	cache           sync.Map // key -> *cacheEntry
}

// NewSettingsConfigService creates a new settings configuration service
func NewSettingsConfigService(settingsService SettingReader) *SettingsConfigService {
	return &SettingsConfigService{
		settingsService: settingsService,
		cache:           sync.Map{},
	}
}

func (s *SettingsConfigService) getFromCache(key string) (interface{}, bool) {
	cached, ok := s.cache.Load(key)
	if !ok {
		return nil, false
	}
	entry, ok := cached.(*cacheEntry)
	if !ok {
		return nil, false
	}
	if clock.Now().After(entry.expiration) {
		s.cache.Delete(key)
		return nil, false
	}
	return entry.value, true
}

// settingsCacheTTL bounds the in-process cache of setting reads; a committed
// mutation invalidates the whole cache immediately (see InvalidateCache).
const settingsCacheTTL = constants.SettingsCacheTTL

func (s *SettingsConfigService) setCache(key string, value interface{}) {
	s.cache.Store(key, &cacheEntry{
		value:      value,
		expiration: clock.Now().Add(settingsCacheTTL),
	})
}

// InvalidateCache clears every cached setting read. Bound at bootstrap so a
// committed settings mutation is authoritative for the next read instead of
// waiting out the cache TTL.
func (s *SettingsConfigService) InvalidateCache() {
	s.cache.Range(func(key, _ interface{}) bool {
		s.cache.Delete(key)
		return true
	})
}

func (s *SettingsConfigService) getBoolSetting(ctx context.Context, key string, defaultValue bool) bool {
	if s.settingsService == nil {
		return defaultValue
	}

	if cached, ok := s.getFromCache(key); ok {
		if v, ok := cached.(bool); ok {
			return v
		}
		return defaultValue
	}

	setting, err := s.settingsService.GetSettingByKey(ctx, key)
	if err != nil {
		// Cache the fallback too (short TTL): the row does not exist until an
		// admin first saves the toggle, and without negative caching every
		// read pays a DB round-trip for the same not-found answer. Staleness
		// is bounded by the TTL.
		observability.GetLogger().Warn("failed to get boolean setting, using default", "key", key, "error", err)
		s.setCache(key, defaultValue)
		return defaultValue
	}

	raw := strings.TrimSpace(setting.GetStringValue())
	if raw == "" {
		// Empty row value behaves like a missing row (default), matching the
		// frontend toggle's parse.
		s.setCache(key, defaultValue)
		return defaultValue
	}
	value := strings.EqualFold(raw, "true")
	s.setCache(key, value)

	return value
}

// GetSubscriptionFeePerEmployeePerKy retrieves the flat subscription fee in
// VND charged per distinct employee per Kỳ (FR-4, PRD BR-1), with caching.
// Returns DefaultSubscriptionFeePerEmployeePerKy (10,000) when the row is
// missing or invalid; negative values are rejected the same way. Billing
// reports snapshot the returned value at generation time (FR-3), so later
// changes never rewrite history.
func (s *SettingsConfigService) GetSubscriptionFeePerEmployeePerKy(ctx context.Context) int64 {
	if cached, ok := s.getFromCache(SettingKeySubscriptionFeePerEmployeePerKyVnd); ok {
		return cached.(int64)
	}

	setting, err := s.settingsService.GetSettingByKey(ctx, SettingKeySubscriptionFeePerEmployeePerKyVnd)
	if err != nil {
		observability.GetLogger().Warn("subscription fee setting missing, using default", "key", SettingKeySubscriptionFeePerEmployeePerKyVnd, "error", err)
		return DefaultSubscriptionFeePerEmployeePerKy
	}

	value, err := setting.GetFloatValue()
	if err != nil || value < 0 || value != float64(int64(value)) {
		observability.GetLogger().Warn("invalid subscription fee setting, using default", "key", SettingKeySubscriptionFeePerEmployeePerKyVnd, "value", value)
		return DefaultSubscriptionFeePerEmployeePerKy
	}

	fee := int64(value)
	s.setCache(SettingKeySubscriptionFeePerEmployeePerKyVnd, fee)
	return fee
}
