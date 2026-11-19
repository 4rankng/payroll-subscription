package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestShiftLabelHourType verifies the rateless-template label mapping: labels
// carrying OT map to overtime, the regular vocabulary maps to "ca ngày", and
// unknown labels are rejected so they still surface the missing-rate error.
func TestShiftLabelHourType(t *testing.T) {
	cases := []struct {
		label string
		want  string
		ok    bool
	}{
		{"CB", "ca ngày", true},
		{"OT", "tăng ca", true},
		{"CN", "ca ngày", true},
		{"OT CN", "tăng ca", true},
		{"cb", "ca ngày", true},
		{"HC", "ca ngày", true},
		{"CB N", "ca ngày", true},
		{"OT Đ", "tăng ca", true},
		// TCN (tăng ca đêm) is a compact code from other templates — substring
		// matching would silently bucket it as regular hours (underpay).
		{"TCN", "", false},
		{"TCNN", "", false},
		{"NN", "", false},
		{"", "", false},
		{"XYZ", "", false},
	}
	for _, tc := range cases {
		got, ok := shiftLabelHourType(tc.label)
		if got != tc.want || ok != tc.ok {
			t.Errorf("shiftLabelHourType(%q) = (%q, %v), want (%q, %v)",
				tc.label, got, ok, tc.want, tc.ok)
		}
	}
}

// TestFlatRatesHaveBucket locks the rateless-fallback membership guard: a
// derived (dayType, hourType) bucket must exist in the flattened payrate
// before the fallback may use it.
func TestFlatRatesHaveBucket(t *testing.T) {
	flatRates := map[string]int{
		"chia chọn.ngày thường.ca ngày": 37500,
		"chia chọn.ngày thường.tăng ca": 56250,
		"lái xe nâng.ngày lễ.ca ngày":   112500,
		"odd.depth.here.ca ngày":        1,
		"chia chọn.ngày nghỉ.ca ngày":   0,
	}
	assert.True(t, flatRatesHaveBucket(flatRates, "ngày thường", "ca ngày"))
	assert.True(t, flatRatesHaveBucket(flatRates, "Ngày Thường", "Tăng Ca"))
	assert.True(t, flatRatesHaveBucket(flatRates, "ngày lễ", "ca ngày"))
	assert.False(t, flatRatesHaveBucket(flatRates, "ngày nghỉ", "ca ngày"))
	assert.False(t, flatRatesHaveBucket(flatRates, "ngày thường", "ca đêm"))
}
