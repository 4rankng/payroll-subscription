package employee

import (
	"context"
	"testing"

	"api-server/internal/domain"
	"api-server/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// Delete rounds out the in-memory fake from request_access_test.go with the
// soft-delete behavior the revoke paths exercise.
func (r *fakeEmployeeUserRepo) Delete(_ context.Context, employeeID, userID uint) error {
	delete(r.rows, [2]uint{employeeID, userID})
	return nil
}

func TestRevokeSelfReleaseDeletesOwnClaimRow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)
	employeeUserRepo := newFakeEmployeeUserRepo()
	employeeUserRepo.rows[[2]uint{31, 44}] = &domain.EmployeeUser{EmployeeID: 31, UserID: 44}
	events := &recordingEventBus{}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		userRepo:         userRepo,
		eventBus:         events,
	}

	// Employee created by partner A (user 9); partner B (user 44) releases
	// their own claim.
	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, Fullname: "Nguyễn Văn A", CreatedBy: 9}, nil)
	userRepo.EXPECT().GetByID(gomock.Any(), uint(44)).Return(&domain.User{ID: 44, Fullname: "Quản lý B", Role: domain.RoleQuanly}, nil)

	err := service.RevokeEmployeeAccess(context.Background(), 31, 44, 44, string(domain.RoleQuanly))
	require.NoError(t, err)
	require.NotContains(t, employeeUserRepo.rows, [2]uint{31, 44}, "own claim row must be deleted")

	require.Len(t, events.events, 1, "real self-release publishes the revoke audit event")
	revoked, ok := events.events[0].(domain.EmployeeAccessRevokedEvent)
	require.True(t, ok)
	require.Equal(t, uint(44), revoked.RevokedUserID, "audit must state whom the revoke removed")
	require.Equal(t, uint(44), revoked.RevokedBy)
}

func TestRevokeSelfReleaseWithoutClaimRowIsSilentNoOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	employeeUserRepo := newFakeEmployeeUserRepo()
	events := &recordingEventBus{}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		eventBus:         events,
	}

	// Project-visible employees can reach a quanly with no claim row at all:
	// releasing then must succeed without a phantom audit event.
	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, CreatedBy: 9}, nil)

	err := service.RevokeEmployeeAccess(context.Background(), 31, 44, 44, string(domain.RoleQuanly))
	require.NoError(t, err)
	require.Empty(t, events.events, "no-row release must not publish a revoke event")
}

func TestRevokeSelfReleaseRejectsNonPartnerRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	employeeUserRepo := newFakeEmployeeUserRepo()
	employeeUserRepo.rows[[2]uint{31, 44}] = &domain.EmployeeUser{EmployeeID: 31, UserID: 44}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		eventBus:         discardEventBus{},
	}

	// Role gate mirrors the claim: only quanly self-releases, so casbin drift
	// can never expose the write path to giamdoc/ketoan.
	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, CreatedBy: 9}, nil)

	err := service.RevokeEmployeeAccess(context.Background(), 31, 44, 44, string(domain.RoleGiamdoc))
	require.Error(t, err, "giamdoc cannot self-release")
	require.NotEmpty(t, employeeUserRepo.rows, "row must survive a rejected release")
}

func TestRevokeByQuanlyOfAnotherUsersAccessForbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	employeeUserRepo := newFakeEmployeeUserRepo()
	employeeUserRepo.rows[[2]uint{31, 55}] = &domain.EmployeeUser{EmployeeID: 31, UserID: 55}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		eventBus:         discardEventBus{},
	}

	// A quanly who is not the creator may never revoke ANOTHER user's access.
	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, CreatedBy: 9}, nil)

	err := service.RevokeEmployeeAccess(context.Background(), 31, 55, 44, string(domain.RoleQuanly))
	require.Error(t, err, "self-release covers own access only")
	require.NotEmpty(t, employeeUserRepo.rows, "target row must survive a rejected revoke")
}

func TestRevokeByCreatorStillAllowed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)
	employeeUserRepo := newFakeEmployeeUserRepo()
	employeeUserRepo.rows[[2]uint{31, 44}] = &domain.EmployeeUser{EmployeeID: 31, UserID: 44}
	events := &recordingEventBus{}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		userRepo:         userRepo,
		eventBus:         events,
	}

	// Creator (user 9) revoking a grant stays allowed and unchanged.
	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, Fullname: "Nguyễn Văn A", CreatedBy: 9}, nil)
	userRepo.EXPECT().GetByID(gomock.Any(), uint(9)).Return(&domain.User{ID: 9, Fullname: "Quản lý A", Role: domain.RoleQuanly}, nil)

	err := service.RevokeEmployeeAccess(context.Background(), 31, 44, 9, string(domain.RoleQuanly))
	require.NoError(t, err)
	require.NotContains(t, employeeUserRepo.rows, [2]uint{31, 44})

	require.Len(t, events.events, 1)
	revoked, ok := events.events[0].(domain.EmployeeAccessRevokedEvent)
	require.True(t, ok)
	require.Equal(t, uint(44), revoked.RevokedUserID)
	require.Equal(t, uint(9), revoked.RevokedBy)
}
