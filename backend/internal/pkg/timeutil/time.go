// Package timeutil provides time helpers.
// All Now() calls delegate to the clock package for testability.
package timeutil

import (
	"time"

	"api-server/internal/pkg/clock"
)

const (
	DateFormat     = "2006-01-02"
	DateTimeFormat = "2006-01-02T15:04:05Z07:00"
	MonthFormat    = "2006-01"
)

// Now returns the current time in the application timezone.

// NowUTC returns the current time in UTC.

// NowUnix returns the current Unix timestamp in seconds.

// TodayStart returns the start of today (00:00:00) in the application timezone.
func TodayStart() time.Time { return clock.Global().TodayStart() }

// TodayEnd returns the end of today (23:59:59.999999999) in the application timezone.

// StartOfDay returns the start of the day (00:00:00) for the given time,
// preserving the input's timezone.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day (23:59:59.999999999) for the given time.
func EndOfDay(t time.Time) time.Time {
	return StartOfDay(t).Add(24*time.Hour - time.Nanosecond)
}

// StartOfMonth returns 00:00:00 on the 1st of the given time's month,
// preserving the input's timezone.
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns 23:59:59.999999999 on the last day of the given time's month.
func EndOfMonth(t time.Time) time.Time {
	return StartOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// StartOfWeek returns 00:00:00 on Monday of the given time's week,
// preserving the input's timezone.

// Sunday = 7

// EndOfWeek returns 23:59:59.999999999 on Sunday of the given time's week.

// StartOfYear returns 00:00:00 on Jan 1 of the given time's year,
// preserving the input's timezone.

// EndOfYear returns 23:59:59.999999999 on Dec 31 of the given time's year.

// DaysAgo returns the time n days before now.

// DaysFromNow returns the time n days after now.

// IsToday reports whether t falls on the current calendar day (VNT).

// clock.Now() — VNT

// IsPast reports whether t is before now.

// IsFuture reports whether t is after now.

// DaysBetween returns the absolute number of days between t1 and t2.

// FormatUnixTimestamp formats a Unix timestamp as "2006-01-02 15:04:05" in UTC.

// ParseBusinessDate parses a date-only value in the application's canonical
// business timezone. Use this for payroll/timesheet calendar dates. ParseDate
// retains its legacy UTC behavior for callers that explicitly need a UTC date.
