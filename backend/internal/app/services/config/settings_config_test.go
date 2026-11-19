package config

import (
	"context"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
)

// keyedStubSettingReader is a SettingsRepository stub that serves canned
// setting rows keyed by name; missing keys surface as not-found errors.
type keyedStubSettingReader struct {
	calls  int
	locks  int
	values map[string]string
}

func (s *keyedStubSettingReader) GetSettingByKey(_ context.Context, key string) (*domain.Settings, error) {
	s.calls++
	value, ok := s.values[key]
	if !ok {
		return nil, domain.NewNotFoundError("setting not found")
	}
	return &domain.Settings{
		Key:       key,
		Value:     new(value),
		ValueType: domain.ValueTypeNumber,
	}, nil
}

func (s *keyedStubSettingReader) GetSettingByKeyAuthoritative(ctx context.Context, key string) (*domain.Settings, error) {
	return s.GetSettingByKey(ctx, key)
}

func (s *keyedStubSettingReader) GetSettingByKeyAuthoritativeForUpdate(ctx context.Context, key string) (*domain.Settings, error) {
	s.locks++
	return s.GetSettingByKey(ctx, key)
}

func feeRow(value string) *domain.Settings {
	return &domain.Settings{
		Key:       SettingKeySubscriptionFeePerEmployeePerKyVnd,
		Value:     new(value),
		ValueType: domain.ValueTypeNumber,
	}
}

func TestSettingsConfigServiceSubscriptionFeeDefaultsWhenRowMissing(t *testing.T) {
	service := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{}})

	assert.Equal(t, int64(DefaultSubscriptionFeePerEmployeePerKy), service.GetSubscriptionFeePerEmployeePerKy(context.Background()))
}

func TestSettingsConfigServiceSubscriptionFeeReadsConfiguredValue(t *testing.T) {
	service := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeySubscriptionFeePerEmployeePerKyVnd: "15000",
	}})

	assert.Equal(t, int64(15000), service.GetSubscriptionFeePerEmployeePerKy(context.Background()))
}

func TestSettingsConfigServiceSubscriptionFeeRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "not a number", value: "abc"},
		{name: "negative", value: "-5"},
		{name: "fractional", value: "10.5"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
				SettingKeySubscriptionFeePerEmployeePerKyVnd: tc.value,
			}})

			assert.Equal(t, int64(DefaultSubscriptionFeePerEmployeePerKy), service.GetSubscriptionFeePerEmployeePerKy(context.Background()))
		})
	}
}

func TestSettingsConfigServiceSubscriptionFeeCachesReads(t *testing.T) {
	reader := &keyedStubSettingReader{values: map[string]string{
		SettingKeySubscriptionFeePerEmployeePerKyVnd: "12000",
	}}
	service := NewSettingsConfigService(reader)

	assert.Equal(t, int64(12000), service.GetSubscriptionFeePerEmployeePerKy(context.Background()))
	assert.Equal(t, 1, reader.calls, "the fee row must be read once and cached")

	assert.Equal(t, int64(12000), service.GetSubscriptionFeePerEmployeePerKy(context.Background()))
	assert.Equal(t, 1, reader.calls, "repeat reads must be served from the cache")
}
