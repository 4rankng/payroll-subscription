package config

// HolidayService handles Vietnam public holiday detection
type HolidayService struct{}

// NewHolidayService creates a new holiday service

// Vietnam public holidays in YYYY-MM-DD format
var holidays = []string{
	"2026-01-01",
	"2026-02-16", "2026-02-17", "2026-02-18", "2026-02-19", "2026-02-20",
	"2026-04-26", "2026-04-27",
	"2026-04-30",
	"2026-05-01",
	"2026-09-01", "2026-09-02",
}

// IsHoliday checks if the given date is a Vietnam public holiday

// GetDayType determines the day type based on Vietnam calendar rules

// Format date as YYYY-MM-DD

// Check if it's a Vietnam public holiday

// Check day of week

// Saturday defaults to "ngày nghỉ" if not specified by user

// IsSaturday checks if the given date is a Saturday
