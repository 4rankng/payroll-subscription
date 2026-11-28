package seed

// isVietnameseHoliday checks if a date is a major Vietnamese public holiday

// EmployeeAttendanceProfile represents individual employee work patterns
type EmployeeAttendanceProfile struct {
	EmployeeID       uint
	Position         string
	AttendanceRate   float64
	IsReliable       bool
	OvertimeTendency float64 // 0.0-1.0, likelihood to work overtime
	WeekendWorkRate  float64 // 0.0-1.0, likelihood to work weekends
}

// Create employee attendance profiles based on positions

// Set attendance characteristics based on position

// 75-90%
// 60% reliable
// 10-30%
// 5-20%

// 80-95%
// 75% reliable
// 15-40%
// 10-30%

// 85-97%
// 85% reliable
// 20-50%
// 15-40%

// 88-98%
// 90% reliable
// 25-60%
// 20-50%

// 90-98%
// 95% reliable
// 30-70%
// 25-60%

// 92-98%
// 95% reliable
// 35-80%
// 30-70%

// Generate timesheets for the past year

// Adjust time range based on project employee assignment dates

// Skip Sundays (Vietnamese work week is Monday-Saturday)

// Seasonal and individual attendance patterns

// Seasonal adjustments

// Tet season - reduced attendance

// Summer vacation season

// Busy manufacturing season

// Year-end - moderate reduction

// Random attendance check

// Public holidays - very low work probability

// Only 15% chance of work on holidays

// Saturday work

// 4-8 hours on Saturday

// Weekday work

// Overtime decision based on individual tendency and seasonal factors

// Q4 busy season

// 1-4 hours overtime

// Slight variation in normal working hours
// ±0.5 hours

// Minimum 6 hours for normal day

// Special holiday pay type

// Round to 2 decimal places

// Realistic timesheet status distribution

// Status depends on reliability and recency

// Old entries are mostly approved

// Recent entries may still be pending

// Very recent entries

// Less reliable employees have more rejections and pending entries
