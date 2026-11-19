package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// The stubs embed their interface so an unexpected call to any other method
// panics, keeping each test honest about which repository surface it touches.
type assignmentUpdateRepoStub struct {
	domain.ProjectEmployeeRepository
	byID        map[uint]*domain.ProjectEmployee
	overlapping []*domain.ProjectEmployee
}

func (s *assignmentUpdateRepoStub) GetByID(_ context.Context, id uint) (*domain.ProjectEmployee, error) {
	if a, ok := s.byID[id]; ok {
		return a, nil
	}
	return nil, domain.NewNotFoundError(constants.MsgAssignmentNotFoundVN)
}

func (s *assignmentUpdateRepoStub) GetOverlappingAssignments(_ context.Context, _ uint, _ *time.Time, _ *time.Time) ([]*domain.ProjectEmployee, error) {
	return s.overlapping, nil
}

type assignmentUpdateEmployeeStub struct {
	domain.EmployeeRepository
}

func (s *assignmentUpdateEmployeeStub) GetByID(_ context.Context, _ uint) (*domain.Employee, error) {
	return &domain.Employee{}, nil
}

type assignmentUpdateProjectStub struct {
	domain.ProjectRepository
}

func (s *assignmentUpdateProjectStub) GetByID(_ context.Context, _ uint) (*domain.Project, error) {
	return &domain.Project{}, nil
}

type assignmentUpdateCheckerStub struct {
	domain.TimesheetAssignmentChecker
	countTimesheets int64
	inRangeResult   bool
}

func (s *assignmentUpdateCheckerStub) CountTimesheets(_ context.Context, _, _ uint) (int64, error) {
	return s.countTimesheets, nil
}

func (s *assignmentUpdateCheckerStub) HasNonEditableTimesheetsInRange(_ context.Context, _, _ uint, _, _ *time.Time) (bool, error) {
	return s.inRangeResult, nil
}

func (s *assignmentUpdateCheckerStub) HasNonEditableTimesheetsAfterDate(_ context.Context, _, _ uint, _ time.Time) (bool, error) {
	panic("HasNonEditableTimesheetsAfterDate must not be called on the update path")
}

func parseDay(s string) time.Time {
	day, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return day
}

func parseDayPtr(s string) *time.Time {
	day := parseDay(s)
	return &day
}

func mkAssignment(id uint, start string) *domain.ProjectEmployee {
	return &domain.ProjectEmployee{
		ID:         id,
		ProjectID:  19,
		EmployeeID: 966,
		StartDate:  parseDay(start),
	}
}

func newUpdateService(repo *assignmentUpdateRepoStub, checker domain.TimesheetAssignmentChecker) *EmployeeAssignmentService {
	return NewEmployeeAssignmentService(repo, &assignmentUpdateEmployeeStub{}, &assignmentUpdateProjectStub{}, checker)
}

func TestValidateAssignmentForUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		current     *domain.ProjectEmployee
		updated     *domain.ProjectEmployee
		checker     *assignmentUpdateCheckerStub
		overlapping []*domain.ProjectEmployee
		wantCode    string
		wantMsg     string
	}{
		{
			name:     "blocks moving start date later over paid timesheets",
			current:  mkAssignment(1, "2026-07-02"),
			updated:  mkAssignment(1, "2026-08-01"),
			checker:  &assignmentUpdateCheckerStub{inRangeResult: true},
			wantCode: "VALIDATION_ERROR",
			wantMsg:  constants.MsgAssignmentUpdateExcludesPaidTimesheetsVN,
		},
		{
			name:    "allows moving start date earlier",
			current: mkAssignment(1, "2026-07-02"),
			updated: mkAssignment(1, "2026-06-15"),
			checker: &assignmentUpdateCheckerStub{},
		},
		{
			name:    "keeps start date unchanged without consulting the guard",
			current: mkAssignment(1, "2026-07-02"),
			updated: mkAssignment(1, "2026-07-02"),
			checker: &assignmentUpdateCheckerStub{},
		},
		{
			name:        "preserves same-project overlap conflict",
			current:     mkAssignment(1, "2026-07-02"),
			updated:     mkAssignment(1, "2026-07-02"),
			checker:     &assignmentUpdateCheckerStub{countTimesheets: 3},
			overlapping: []*domain.ProjectEmployee{mkAssignment(2, "2026-08-01")},
			wantCode:    "CONFLICT",
			wantMsg:     constants.MsgAssignmentOverlapVN,
		},
		{
			name:    "ignores overlapping assignment on a different project",
			current: mkAssignment(1, "2026-07-02"),
			updated: mkAssignment(1, "2026-07-02"),
			checker: &assignmentUpdateCheckerStub{countTimesheets: 3},
			overlapping: []*domain.ProjectEmployee{
				{ID: 2, ProjectID: 68, EmployeeID: 966, StartDate: parseDay("2026-08-01")},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &assignmentUpdateRepoStub{
				byID:        map[uint]*domain.ProjectEmployee{1: tc.current},
				overlapping: tc.overlapping,
			}
			svc := newUpdateService(repo, tc.checker)

			err := svc.ValidateAssignmentForUpdate(context.Background(), tc.updated)

			if tc.wantCode == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			domainErr, ok := err.(*domain.DomainError)
			require.True(t, ok, "expected *domain.DomainError, got %T", err)
			require.Equal(t, tc.wantCode, domainErr.Code)
			require.True(t, strings.Contains(domainErr.Message, tc.wantMsg))
		})
	}
}

// ---------------------------------------------------------------------------
// Window bounds: the guard must query the checker ONLY when the start actually
// moves later, and with exact day-granular bounds. An off-by-one here is
// invisible to the table above (its stub ignores the dates), so these tests
// capture and assert the from/to arguments verbatim.
// ---------------------------------------------------------------------------

type rangeBoundsQuery struct{ from, to *time.Time }

func formatDay(p *time.Time) string {
	if p == nil {
		return "<nil>"
	}
	return p.Format("2006-01-02")
}

// boundsCapturingChecker embeds the table-test stub (so CountTimesheets and
// the AfterDate panic-guard keep working) and records every range query.
type boundsCapturingChecker struct {
	assignmentUpdateCheckerStub
	inRange bool
	err     error
	calls   []rangeBoundsQuery
}

func (s *boundsCapturingChecker) HasNonEditableTimesheetsInRange(_ context.Context, _, _ uint, from, to *time.Time) (bool, error) {
	s.calls = append(s.calls, rangeBoundsQuery{from: from, to: to})
	if s.err != nil {
		return false, s.err
	}
	return s.inRange, nil
}

func TestValidateAssignmentForUpdateWindowBounds(t *testing.T) {
	// Employee-966 shape: window opens 2026-07-02, weekly pay.
	current := mkAssignment(1, "2026-07-02")

	run := func(updated *domain.ProjectEmployee, checker *boundsCapturingChecker) error {
		repo := &assignmentUpdateRepoStub{byID: map[uint]*domain.ProjectEmployee{1: current}}
		svc := newUpdateService(repo, checker)
		return svc.ValidateAssignmentForUpdate(context.Background(), updated)
	}

	t.Run("start moved later queries old start through day before new start", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		require.NoError(t, run(mkAssignment(1, "2026-08-01"), checker))
		require.Len(t, checker.calls, 1, "exactly one range query expected")
		require.Equal(t, "2026-07-02", formatDay(checker.calls[0].from), "from must be the old start")
		require.Equal(t, "2026-07-31", formatDay(checker.calls[0].to), "to must be the day before the new start")
	})

	t.Run("start moved earlier queries nothing", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		require.NoError(t, run(mkAssignment(1, "2026-06-15"), checker))
		require.Empty(t, checker.calls)
	})

	t.Run("start unchanged queries nothing", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		require.NoError(t, run(mkAssignment(1, "2026-07-02"), checker))
		require.Empty(t, checker.calls, "keeping the start date must not consult the paid-timesheet guard")
	})

	t.Run("checker error propagates", func(t *testing.T) {
		boom := errors.New("db down")
		checker := &boundsCapturingChecker{err: boom}
		err := run(mkAssignment(1, "2026-08-01"), checker)
		require.ErrorIs(t, err, boom)
	})
}

// ---------------------------------------------------------------------------
// Pure guard directly: overlap variants the impure table cannot express.
// ---------------------------------------------------------------------------

func TestValidateAssignmentUpdateWithDataOverlapVariants(t *testing.T) {
	svc := NewEmployeeAssignmentService(nil, nil, nil, nil)
	updated := mkAssignment(1, "2026-07-02")

	t.Run("skips the assignment itself", func(t *testing.T) {
		self := mkAssignment(1, "2026-07-02")
		require.NoError(t, svc.ValidateAssignmentUpdateWithData(updated, []*domain.ProjectEmployee{self}, map[uint]int64{1: 5}, false))
	})

	t.Run("allows same-project overlap when the other assignment has no timesheets", func(t *testing.T) {
		other := mkAssignment(2, "2026-08-01")
		require.NoError(t, svc.ValidateAssignmentUpdateWithData(updated, []*domain.ProjectEmployee{other}, map[uint]int64{2: 0}, false))
	})

	t.Run("rejects when the start window is dirty", func(t *testing.T) {
		err := svc.ValidateAssignmentUpdateWithData(updated, nil, nil, true)
		require.Error(t, err)
		domainErr, ok := err.(*domain.DomainError)
		require.True(t, ok)
		require.Equal(t, "VALIDATION_ERROR", domainErr.Code)
	})
}

// ---------------------------------------------------------------------------
// Base validation preserved: early rejects never reach the paid-timesheet guard.
// ---------------------------------------------------------------------------

type notFoundEmployeeStub struct {
	domain.EmployeeRepository
}

func (notFoundEmployeeStub) GetByID(_ context.Context, _ uint) (*domain.Employee, error) {
	return nil, domain.NewNotFoundError(constants.MsgEmployeeNotFoundVN)
}

func TestValidateAssignmentForUpdateBaseValidation(t *testing.T) {
	t.Run("zero start date rejected before any range query", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		repo := &assignmentUpdateRepoStub{byID: map[uint]*domain.ProjectEmployee{1: mkAssignment(1, "2026-07-02")}}
		svc := newUpdateService(repo, checker)
		err := svc.ValidateAssignmentForUpdate(context.Background(), &domain.ProjectEmployee{ID: 1})
		require.Error(t, err)
		domainErr, ok := err.(*domain.DomainError)
		require.True(t, ok)
		require.Equal(t, "VALIDATION_ERROR", domainErr.Code)
		require.Empty(t, checker.calls, "dates reject must fire before the paid-timesheet guard")
	})

	t.Run("missing employee surfaces as not found", func(t *testing.T) {
		svc := NewEmployeeAssignmentService(
			&assignmentUpdateRepoStub{},
			notFoundEmployeeStub{},
			&assignmentUpdateProjectStub{},
			&assignmentUpdateCheckerStub{},
		)
		err := svc.ValidateAssignmentForUpdate(context.Background(), mkAssignment(1, "2026-07-02"))
		require.Error(t, err)
		domainErr, ok := err.(*domain.DomainError)
		require.True(t, ok)
		require.Equal(t, "NOT_FOUND", domainErr.Code)
	})
}

// The rule: last recorded timesheet + 1 day when the employee has history,
// else the 1st of the current month — never "today", which would reject
// same-month BCC uploads covering earlier days (2026-09-17 CBS incident).
func TestSuggestAssignmentStart(t *testing.T) {
	now := time.Date(2026, 9, 17, 15, 0, 0, 0, time.UTC)

	cases := []struct {
		name   string
		latest *time.Time
		want   time.Time
	}{
		{"no history falls back to month start", nil, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
		{"mid-month history continues next day", parseDayPtr("2026-09-10"), time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)},
		{"history equals today starts tomorrow", parseDayPtr("2026-09-17"), time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)},
		{"previous-month history still continues next day", parseDayPtr("2026-08-31"), time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SuggestAssignmentStart(tc.latest, now)
			if !got.Equal(tc.want) {
				t.Fatalf("SuggestAssignmentStart(%v, %v) = %v, want %v", tc.latest, now, got, tc.want)
			}
		})
	}
}
