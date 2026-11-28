package clock

import (
	"testing"
)

// TestPeriodCycleStartDay guards the period rollover: the advance-payment
// period rolls on day 20. The request tail length is Admin-configurable
// (advance_request_stop_day, default 9), so it is no longer a constant here.
func TestPeriodCycleStartDay(t *testing.T) {
	if PeriodCycleStartDay != 20 {
		t.Fatalf("PeriodCycleStartDay = %d, want 20", PeriodCycleStartDay)
	}
}

// TestRequestWindowBoundaries exercises the three-phase window around the
// default day-9 stop day using Asia/Ho_Chi_Minh local dates (the timezone
// the business runs in). July 2026 anchors the phases relative to calendar
// July:
//
//	days 1–8  -> tail of the previous period (for_month = June)
//	days 9–19 -> locked gap
//	days 20+  → new period opens (for_month rolls to July/August)

// final tail day, still previous period
// stop day, requests locked
// locked gap
// mid locked gap
// last locked-gap day
// new period opens

// A configured stop day shifts the boundary: with stop day 12, day 10 stays
// requestable and day 12 is the first locked day while the gap still ends
// at day 20.

// inside a stop day 12 tail
// last tail day under stop day 12
// stop day, requests locked
// gap still ends before day 20
// new period opens
