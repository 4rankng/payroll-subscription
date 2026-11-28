package config

import (
	"context"
	"strings"
	"testing"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/infra/persistence"
	"api-server/internal/pkg/secret"

	"github.com/alicebob/miniredis/v2"
)

type fakeSettingsEventBus struct{}

func (fakeSettingsEventBus) Publish(context.Context, ...domain.DomainEvent) error { return nil }
func (fakeSettingsEventBus) Subscribe(string, domain.EventHandler)                {}
func (fakeSettingsEventBus) SubscribeAll(domain.EventHandler)                     {}

type fakeSettingsRepoForUpdate struct {
	rows map[uint]*domain.Settings
}

func newFakeSettingsRepoForUpdate(rows ...*domain.Settings) *fakeSettingsRepoForUpdate {
	out := &fakeSettingsRepoForUpdate{rows: make(map[uint]*domain.Settings, len(rows))}
	for _, row := range rows {
		cp := *row
		out.rows[row.ID] = &cp
	}
	return out
}

func (r *fakeSettingsRepoForUpdate) snapshot() map[uint]*domain.Settings {
	out := make(map[uint]*domain.Settings, len(r.rows))
	for id, row := range r.rows {
		cp := *row
		out[id] = &cp
	}
	return out
}

func (r *fakeSettingsRepoForUpdate) restore(rows map[uint]*domain.Settings) {
	r.rows = rows
}

func (r *fakeSettingsRepoForUpdate) Create(context.Context, *domain.Settings) error {
	panic("unexpected Create call")
}

func (r *fakeSettingsRepoForUpdate) GetByID(_ context.Context, id uint) (*domain.Settings, error) {
	row, ok := r.rows[id]
	if !ok {
		return nil, domain.NewNotFoundError("setting not found")
	}
	cp := *row
	return &cp, nil
}

func (r *fakeSettingsRepoForUpdate) GetByIDForUpdate(ctx context.Context, id uint) (*domain.Settings, error) {
	return r.GetByID(ctx, id)
}

func (r *fakeSettingsRepoForUpdate) GetByKey(_ context.Context, key string) (*domain.Settings, error) {
	for _, row := range r.rows {
		if row.Key == key {
			cp := *row
			return &cp, nil
		}
	}
	return nil, domain.NewNotFoundError("setting not found")
}

func (r *fakeSettingsRepoForUpdate) GetByKeyForUpdate(ctx context.Context, key string) (*domain.Settings, error) {
	return r.GetByKey(ctx, key)
}

func (r *fakeSettingsRepoForUpdate) Update(_ context.Context, setting *domain.Settings) error {
	cp := *setting
	r.rows[setting.ID] = &cp
	return nil
}

func (r *fakeSettingsRepoForUpdate) CompareAndSwapValue(_ context.Context, key, currentValue, nextValue string, valueType domain.SettingsValueType) (bool, error) {
	for id, setting := range r.rows {
		if setting.Key != key || setting.Value == nil || *setting.Value != currentValue {
			continue
		}
		copySetting := *setting
		copySetting.Value = &nextValue
		copySetting.ValueType = valueType
		r.rows[id] = &copySetting
		return true, nil
	}
	return false, nil
}

// EncryptProtectedValues is a no-op here: the fake stores values in memory and
// these tests exercise the update flow, not the startup secret backfill (that
// lives in the persistence layer, covered by settings_repository_secrets_test).
func (r *fakeSettingsRepoForUpdate) EncryptProtectedValues(context.Context) ([]string, error) {
	return nil, nil
}

func (r *fakeSettingsRepoForUpdate) Delete(context.Context, uint) error {
	panic("unexpected Delete call")
}

func (r *fakeSettingsRepoForUpdate) List(context.Context, domain.SettingsFilters) ([]*domain.Settings, error) {
	panic("unexpected List call")
}

func (r *fakeSettingsRepoForUpdate) Count(context.Context, domain.SettingsFilters) (int64, error) {
	panic("unexpected Count call")
}

type fakeSettingsTxManager struct {
	settings *fakeSettingsRepoForUpdate
}

func (tm fakeSettingsTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	settingsSnapshot := tm.settings.snapshot()
	if err := fn(ctx); err != nil {
		tm.settings.restore(settingsSnapshot)
		return err
	}
	return nil
}

func (tm fakeSettingsTxManager) WithTransactionResult(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	settingsSnapshot := tm.settings.snapshot()
	result, err := fn(ctx)
	if err != nil {
		tm.settings.restore(settingsSnapshot)
	}
	return result, err
}

func TestSettingsServiceUpdateSettingNullValueSemantics(t *testing.T) {
	const credential = `{"app_id":"123","secret_key":"s3cr3t-key"}`

	settingsRepo := newFakeSettingsRepoForUpdate(
		&domain.Settings{ID: 1, Key: secret.ZaloCredentialsKey, Value: new(credential), ValueType: domain.ValueTypeJSON},
		&domain.Settings{ID: 2, Key: "transfer_bank_visible", Value: new("true"), ValueType: domain.ValueTypeString},
	)
	service := NewSettingsService(
		settingsRepo,
		fakeSettingsTxManager{settings: settingsRepo},
		nil,
		fakeSettingsEventBus{},
	)

	protected, err := service.UpdateSetting(context.Background(), 1, map[string]any{"value": nil}, 7)
	if err != nil {
		t.Fatalf("UpdateSetting(protected, null): %v", err)
	}
	if protected.Value == nil || *protected.Value != credential {
		t.Fatalf("protected value = %v, want the stored credential to survive a null update", protected.Value)
	}

	ordinary, err := service.UpdateSetting(context.Background(), 2, map[string]any{"value": nil}, 7)
	if err != nil {
		t.Fatalf("UpdateSetting(ordinary, null): %v", err)
	}
	if ordinary.Value != nil {
		t.Fatalf("ordinary value = %v, want the null update to clear it", *ordinary.Value)
	}
}

// The settings cache is Redis; caching a decrypted credential would recreate the
// plaintext exposure the repository now seals in MySQL.
func TestSettingsServiceDoesNotCacheProtectedSetting(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb, err := persistence.NewRedisClient(persistence.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatalf("connect miniredis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	settingsRepo := newFakeSettingsRepoForUpdate(
		&domain.Settings{ID: 1, Key: secret.ZaloCredentialsKey, Value: new(`{"app_id":"123"}`), ValueType: domain.ValueTypeJSON},
		&domain.Settings{ID: 2, Key: "transfer_bank_visible", Value: new("true"), ValueType: domain.ValueTypeString},
	)
	service := NewSettingsService(
		settingsRepo,
		nil,
		infrastructure.NewCacheService(rdb),
		fakeSettingsEventBus{},
	)

	if _, err := service.GetSettingByKey(context.Background(), "transfer_bank_visible"); err != nil {
		t.Fatalf("GetSettingByKey(ordinary): %v", err)
	}
	if _, err := service.GetSettingByKey(context.Background(), secret.ZaloCredentialsKey); err != nil {
		t.Fatalf("GetSettingByKey(protected): %v", err)
	}

	keys := mr.Keys()
	if len(keys) != 1 || !strings.Contains(keys[0], "transfer_bank_visible") {
		t.Fatalf("cached keys = %v, want only the ordinary setting", keys)
	}
}

type fakeConfigCacheInvalidator struct{ calls int }

func (f *fakeConfigCacheInvalidator) InvalidateCache() { f.calls++ }

// A committed update must clear the in-process SettingsConfigService cache too:
// subscription-fee reads go through it, and the row-level Redis invalidation
// above does not reach that cache.
func TestSettingsServiceUpdateSettingInvalidatesConfigCache(t *testing.T) {
	settingsRepo := newFakeSettingsRepoForUpdate(&domain.Settings{
		ID:        1,
		Key:       SettingKeySubscriptionFeePerEmployeePerKyVnd,
		Value:     new("false"),
		ValueType: domain.ValueTypeString,
	})
	service := NewSettingsService(
		settingsRepo,
		fakeSettingsTxManager{settings: settingsRepo},
		nil,
		fakeSettingsEventBus{},
	)

	invalidator := &fakeConfigCacheInvalidator{}
	service.SetConfigCacheInvalidator(invalidator)

	if _, err := service.UpdateSetting(context.Background(), 1, map[string]any{"value": "true"}, 7); err != nil {
		t.Fatalf("UpdateSetting returned error: %v", err)
	}
	if invalidator.calls != 1 {
		t.Fatalf("bound config cache invalidator calls = %d, want 1", invalidator.calls)
	}
}
