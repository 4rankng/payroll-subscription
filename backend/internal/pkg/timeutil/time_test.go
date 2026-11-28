package timeutil

import (
	"testing"
	"time"
)

func TestTodayStart(t *testing.T) {
	start := TodayStart()
	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Errorf("TodayStart() did not return start of day: %v", start)
	}
}

func TestStartOfDay(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	start := StartOfDay(testTime)

	expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !start.Equal(expected) {
		t.Errorf("StartOfDay() = %v, want %v", start, expected)
	}
}

func TestEndOfDay(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	end := EndOfDay(testTime)

	// Check that it's the last nanosecond of the day
	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Errorf("EndOfDay() did not return end of day: %v", end)
	}
}

// Should be approximately 7 days (within a small margin)

// Should be approximately 7 days (within a small margin)

// March 15, 2024 is a Friday

// Monday March 11, 2024 00:00:00

func TestStartOfMonth(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	start := StartOfMonth(testTime)

	expected := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(expected) {
		t.Errorf("StartOfMonth() = %v, want %v", start, expected)
	}
}

func TestEndOfMonth(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	end := EndOfMonth(testTime)

	// March has 31 days
	if end.Day() != 31 || end.Month() != time.March || end.Year() != 2024 {
		t.Errorf("EndOfMonth() = %v, expected last day of March 2024", end)
	}
}

// Test reverse order (should still be positive)

// Unix timestamp for 2024-03-15 14:30:45 UTC
