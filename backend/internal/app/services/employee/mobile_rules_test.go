package employee

import (
	"context"
	"testing"

	userservice "api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// mobileRulesRepo layers a controllable mobile lookup on the shared in-memory
// stub so the create/update mobile rules can be exercised in isolation.
type mobileRulesRepo struct {
	stubEmployeeRepo
	holdersByMobile map[string][]*domain.Employee
}

func (r *mobileRulesRepo) ListByMobile(_ context.Context, mobile string) ([]*domain.Employee, error) {
	return r.holdersByMobile[mobile], nil
}

// newMobileRulesService wires the same minimal harness the create tests use
// (create_after_commit_test.go): direct transactions, no-op events, in-memory
// user repo.
func newMobileRulesService(repo *mobileRulesRepo) *EmployeeService {
	users := &createEmployeeTestUserRepo{}
	events := createEmployeeTestEventBus{}
	return &EmployeeService{
		EmployeeRepo:       repo,
		UserRepo:           users,
		UserService:        userservice.NewUserService(users, nil, events, "test-secret", "test-salt", nil),
		TransactionManager: directTransactionManager{},
		events:             events,
	}
}


// Dropping the mandatory-mobile policy: a phone-less employee is
// legal on every create path. Pins the acceptance behavior so a silent
// re-introduction of the requirement cannot pass CI.
func TestCreateEmployee_MissingMobileAllowed(t *testing.T) {
	service := newMobileRulesService(&mobileRulesRepo{})

	t.Run("manual path creates without a mobile", func(t *testing.T) {
		created, err := service.CreateEmployee(context.Background(), &domain.Employee{Fullname: "Nguyen Van A", CCCD: "012345678901"}, 9)
		require.NoError(t, err)
		require.Empty(t, created.Mobile)
	})

	t.Run("import path shares createEmployeeCore and creates without a mobile", func(t *testing.T) {
		_, err := service.CreateEmployeeFromImport(context.Background(), &domain.Employee{Fullname: "Nguyen Van A", CCCD: "012345678901"}, 9)
		require.NoError(t, err)
	})
}

func TestCreateEmployee_RejectsDuplicateMobile(t *testing.T) {
	repo := &mobileRulesRepo{holdersByMobile: map[string][]*domain.Employee{
		"0912345678": {{ID: 7, Fullname: "Existing Holder", CCCD: "001179010398"}},
	}}
	service := newMobileRulesService(repo)

	t.Run("explicit collision is a conflict", func(t *testing.T) {
		emp := &domain.Employee{Fullname: "Nguyen Van B", CCCD: "012345678902", Mobile: "0912345678"}
		_, err := service.CreateEmployee(context.Background(), emp, 9)
		require.ErrorContains(t, err, constants.MsgEmployeeMobileTakenVN)
	})

	t.Run("outer spaces do not evade the lookup", func(t *testing.T) {
		emp := &domain.Employee{Fullname: "Nguyen Van B", CCCD: "012345678902", Mobile: " 0912345678 "}
		_, err := service.CreateEmployee(context.Background(), emp, 9)
		require.ErrorContains(t, err, constants.MsgEmployeeMobileTakenVN)
	})

	t.Run("a free number creates", func(t *testing.T) {
		emp := &domain.Employee{Fullname: "Nguyen Van B", CCCD: "012345678902", Mobile: "0987654321"}
		_, err := service.CreateEmployee(context.Background(), emp, 9)
		require.NoError(t, err)
	})
}

func TestUpdateEmployee_MobileDuplicateSelfExclusion(t *testing.T) {
	base := func() *domain.Employee {
		return &domain.Employee{ID: 1, Fullname: "Nguyen Van A", CCCD: "012345678901", Mobile: "0911111111"}
	}

	t.Run("moving onto a number someone else holds is rejected", func(t *testing.T) {
		repo := &mobileRulesRepo{
			stubEmployeeRepo: stubEmployeeRepo{byID: map[uint]*domain.Employee{1: base()}},
			holdersByMobile: map[string][]*domain.Employee{
				"0922222222": {{ID: 2, Fullname: "Other", CCCD: "012345678902"}},
			},
		}
		service := newMobileRulesService(repo)

		emp := base()
		emp.Mobile = "0922222222"
		err := service.UpdateEmployee(context.Background(), emp, 9)
		require.ErrorContains(t, err, constants.MsgEmployeeMobileTakenVN)
	})

	t.Run("re-saving the same number passes even when a legacy twin holds it", func(t *testing.T) {
		repo := &mobileRulesRepo{
			stubEmployeeRepo: stubEmployeeRepo{byID: map[uint]*domain.Employee{1: base()}},
			holdersByMobile: map[string][]*domain.Employee{
				"0911111111": {
					{ID: 1, Fullname: "Nguyen Van A", CCCD: "012345678901"},
					{ID: 3, Fullname: "Legacy Twin", CCCD: "012345678903"},
				},
			},
		}
		service := newMobileRulesService(repo)

		require.NoError(t, service.UpdateEmployee(context.Background(), base(), 9))
		require.Equal(t, 1, repo.updateCalls)
	})

	t.Run("a number held only by the employee itself passes", func(t *testing.T) {
		repo := &mobileRulesRepo{
			stubEmployeeRepo: stubEmployeeRepo{byID: map[uint]*domain.Employee{1: base()}},
			holdersByMobile: map[string][]*domain.Employee{
				"0922222222": {{ID: 1, Fullname: "Nguyen Van A", CCCD: "012345678901"}},
			},
		}
		service := newMobileRulesService(repo)

		emp := base()
		emp.Mobile = "0922222222"
		require.NoError(t, service.UpdateEmployee(context.Background(), emp, 9))
	})
}

// duplicateGroupsRepo serves the two reads the duplicate-mobile report needs.
type duplicateGroupsRepo struct {
	stubEmployeeRepo
	groups  []*domain.DuplicateIdentityGroup
	holders map[string][]*domain.Employee
}

func (r *duplicateGroupsRepo) GetDuplicateIdentities(context.Context) ([]*domain.DuplicateIdentityGroup, error) {
	return r.groups, nil
}

func (r *duplicateGroupsRepo) ListByMobiles(_ context.Context, mobiles []string) ([]*domain.Employee, error) {
	var out []*domain.Employee
	for _, mobile := range mobiles {
		out = append(out, r.holders[mobile]...)
	}
	return out, nil
}

func TestGetDuplicateMobileGroups_ScopesVisibleMembers(t *testing.T) {
	repo := &duplicateGroupsRepo{
		groups: []*domain.DuplicateIdentityGroup{
			{Field: "mobile", Value: "0912345678", EmployeeIDs: []uint{1, 2}, Count: 2},
			// A CCCD collision is not a mobile group — the nudge only reports numbers.
			{Field: "cccd", Value: "012345678901", EmployeeIDs: []uint{1, 3}, Count: 2},
		},
		holders: map[string][]*domain.Employee{
			"0912345678": {
				{ID: 1, Fullname: "Visible Holder", Mobile: "0912345678"},
				{ID: 2, Fullname: "Hidden Holder", Mobile: "0912345678"},
			},
		},
	}
	service := &EmployeeService{EmployeeRepo: repo}

	t.Run("unscoped sees every holder", func(t *testing.T) {
		groups, err := service.GetDuplicateMobileGroups(context.Background(), nil)
		require.NoError(t, err)
		require.Len(t, groups, 1)
		require.Equal(t, "0912345678", groups[0].Mobile)
		require.Len(t, groups[0].Employees, 2)
		require.Zero(t, groups[0].HiddenCount)
	})

	t.Run("scoped hides out-of-scope holders but keeps the group", func(t *testing.T) {
		groups, err := service.GetDuplicateMobileGroups(context.Background(), []uint{1})
		require.NoError(t, err)
		require.Len(t, groups, 1)
		require.Equal(t, []DuplicateMobileMember{{ID: 1, Fullname: "Visible Holder"}}, groups[0].Employees)
		require.Equal(t, 1, groups[0].HiddenCount)
	})

	t.Run("a group with no visible member is dropped", func(t *testing.T) {
		groups, err := service.GetDuplicateMobileGroups(context.Background(), []uint{99})
		require.NoError(t, err)
		require.Empty(t, groups)
	})
}
