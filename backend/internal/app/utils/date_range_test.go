package utils

// The from-day is a day of the PREVIOUS month; a day missing there (Feb 29 in a
// non-leap year) rolls forward to the next existing day.

// March 2026 cycle: from=29 -> Feb 29 2026 does not exist -> Mar 1 2026.

// In a leap year the 29th exists, so the start day is used as-is.

// A missing end day clamps to the month's last day instead of bleeding into the
// next cycle's days ("cuối tháng" semantics of the 0 sentinel).

// 0 sentinels: from=0 starts at the 1st, to=0 ends at month end.
