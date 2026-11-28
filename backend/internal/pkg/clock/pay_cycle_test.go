package clock

import (
	"testing"
	"time"
)

func TestKyFromWorkDay(t *testing.T) {
	cases := []struct {
		day  int
		want int
	}{
		{1, 1}, {7, 1},
		{8, 2}, {14, 2},
		{15, 3}, {21, 3},
		{22, 4}, {28, 4}, {29, 4}, {31, 4},
	}
	for _, c := range cases {
		if got := KyFromWorkDay(c.day); got != c.want {
			t.Errorf("KyFromWorkDay(%d) = %d, want %d", c.day, got, c.want)
		}
	}
}

// 31-day July: (31-22+1)+1 = 11
// 30-day June: (30-22+1)+1 = 10
// 28-day Feb: (28-22+1)+1 = 8
// leap Feb: (29-22+1)+1 = 9

func TestPayDate(t *testing.T) {
	july := time.Date(2026, time.July, 10, 0, 0, 0, 0, DefaultLocation)
	if got := PayDate(1, 2026, time.July); !got.Equal(july) {
		t.Errorf("PayDate(1, Jul) = %v, want %v", got, july)
	}
	// Ky 4 crosses the month boundary: work July → pay day 1 of August.
	aug1 := time.Date(2026, time.August, 1, 0, 0, 0, 0, DefaultLocation)
	if got := PayDate(4, 2026, time.July); !got.Equal(aug1) {
		t.Errorf("PayDate(4, Jul) = %v, want %v (Aug 1)", got, aug1)
	}
	// Ky 4 year-boundary: work December → pay day 1 of January next year.
	jan1 := time.Date(2027, time.January, 1, 0, 0, 0, 0, DefaultLocation)
	if got := PayDate(4, 2026, time.December); !got.Equal(jan1) {
		t.Errorf("PayDate(4, Dec) = %v, want %v (Jan 1 2027)", got, jan1)
	}
}

// Ky 2 (work-start day 8) approval on day 11 → cycle-day 4.

// Approval before the work window clamps to cycle-day 1.

// Approval after the pay date clamps to MaxCycleDay.

// Ky 4 month-boundary: work-start Jul 22, approval Aug 1 → cycle-day 11.

// Ky 4 approval still within July (Jul 29) → cycle-day 8.

// A timestamp at 23:00 UTC on Jul 11 is 06:00 +07:00 on Jul 12 — the same
// instant must map to cycle-day 5 regardless of which timezone the caller
// passes, because approval is interpreted in DefaultLocation.

// On a cycle's pay day the card must roll FORWARD: that cycle's transfer
// is happening and its prep window (pay date − lead) has closed.

// Ky4 pay date must land in the next calendar month.

// The primary correctness risk: Ky 4 work window Jul 22–31, pay Aug 1.

// cycle-day 11 = the Aug 1 pay date.

// Ky1, pay Jul 10

// dateAt builds a midnight DefaultLocation time for the given calendar date.
func dateAt(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, DefaultLocation)
}

// TestNextPayCycleAfter_WrapAround exercises the helper the settlement
// simulation uses to project "current + next N" cycles. Kỳ 4 must wrap to
// Kỳ 1 of the next month; Kỳ 1→2→3 stay in the same month.

// Kỳ 1, July

// Wrap: Kỳ 4 of July → Kỳ 1 of August.

// And the August Kỳ 1 pays on Aug 10.

// TestCycleWindow covers each Kỳ's work-day window, including the Kỳ 4
// "rest-of-month" case (July has 31 days → days 22–31).
func TestCycleWindow(t *testing.T) {
	cases := []struct {
		ky      int
		fromDay int
		toDay   int
	}{
		{1, 1, 7},
		{2, 8, 14},
		{3, 15, 21},
		{4, 22, 31}, // July has 31 days
	}
	month := dateAt(2026, time.July, 1)
	for _, c := range cases {
		pc := TimesheetPayCycle{Ky: c.ky, WorkMonth: month}
		from, to := pc.CycleWindow()
		if from.Day() != c.fromDay {
			t.Errorf("Kỳ %d from day = %d, want %d", c.ky, from.Day(), c.fromDay)
		}
		if to.Day() != c.toDay {
			t.Errorf("Kỳ %d to day = %d, want %d", c.ky, to.Day(), c.toDay)
		}
	}
}
