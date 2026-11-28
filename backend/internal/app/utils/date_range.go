package utils

// CalculateDateRange determines the start and end dates of a salary period
// based on a reference date (atDate) and cycle configuration.

// 1. Determine Anchor Month using day-of-month heuristic.
// Early month (day ≤ 15): anchor to previous month (previous cycle just ended).
// Late month (day > 15): anchor to current month (current cycle is running/complete).
// This applies uniformly to all period types including weekly (salaryPeriodTo == 0).

// 2. Calculate fromDate

// Custom cycle (e.g. start on the 20th).
// The start date is in the month PRIOR to the Anchor Month.
// Example: Anchor Nov, Start Day 20 -> Oct 20

// Standard cycle (Start of month).
// The start date is the 1st of the Anchor Month.
// Example: Anchor Nov -> Nov 1

// 3. Calculate toDate

// Custom cycle (e.g. end on the 21st).
// The end date is in the Anchor Month. A day past the month's length
// clamps to that month's last day ("cuối tháng" — the 0 sentinel's
// meaning): rolling it forward would pull days belonging to the next
// cycle into this one.

// Standard cycle (End of month).
// The end date is the last day of the Anchor Month.
// Trick: Set date to the 1st of the NEXT month, then subtract 1 day.
