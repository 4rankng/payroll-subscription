package user

import (
	"context"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRoleRepo records which write path UpdateUser took. It embeds the
// interface so only the exercised methods have behaviour; anything else
// panics, which is itself the assertion.
type fakeRoleRepo struct {
	domain.UserRepository

	user *domain.User

	fullRowSaves  int
	combinedSaves int
	invalidBefore []time.Time
}

func (r *fakeRoleRepo) GetByID(_ context.Context, id uint) (*domain.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, domain.NewNotFoundError("user not found")
	}
	c := *r.user
	return &c, nil
}

// Get is reached by auditservice.GetActorFullName when resolving the actor.
func (r *fakeRoleRepo) Get(ctx context.Context, id uint) (*domain.User, error) {
	return r.GetByID(ctx, id)
}

func (r *fakeRoleRepo) Update(_ context.Context, user *domain.User) error {
	r.fullRowSaves++
	r.user = user
	return nil
}

func (r *fakeRoleRepo) UpdateUserAndInvalidateSessions(_ context.Context, user *domain.User, invalidBefore time.Time) error {
	r.combinedSaves++
	r.invalidBefore = append(r.invalidBefore, invalidBefore)
	r.user = user
	return nil
}

type fakeRoleEventBus struct {
	domain.EventBus
}

func (b *fakeRoleEventBus) Publish(_ context.Context, _ ...domain.DomainEvent) error {
	return nil
}

// TestUpdateUser_RoleChange_InvalidatesSessions pins the PAYSUB-32 contract:
// the moment an admin switches a user's role, every existing session dies with
// the old role (old tabs are forced to re-login) and the auth layer's cached
// token verdict is dropped so the rejection lands on the very next request.
func TestUpdateUser_RoleChange_InvalidatesSessions(t *testing.T) {
	fakeNow := time.Date(2026, 11, 28, 12, 0, 0, 0, clock.DefaultLocation)
	repo := &fakeRoleRepo{user: &domain.User{ID: 7, Username: "cuongnv", Fullname: "Cuong", Role: domain.RoleQuanly}}
	svc := NewUserService(repo, nil, &fakeRoleEventBus{}, "test-secret", "test-salt", clock.NewFake(fakeNow))

	var notified []uint
	svc.OnSessionsInvalidated = func(_ context.Context, userID uint) {
		notified = append(notified, userID)
	}

	role := new("nhanvien")
	resp, err := svc.UpdateUser(context.Background(), 7, dto.UpdateUserRequest{Role: role})
	require.NoError(t, err)
	assert.Equal(t, "nhanvien", resp.Role)

	assert.Equal(t, 1, repo.combinedSaves, "a role change must use the atomic save+invalidate path")
	assert.Equal(t, 0, repo.fullRowSaves, "the plain full-row save must not be used on a role change")
	require.Len(t, repo.invalidBefore, 1, "tokens_invalid_before must be advanced")
	assert.True(t, repo.invalidBefore[0].Equal(fakeNow), "invalidBefore must carry the change instant")
	assert.Equal(t, []uint{7}, notified, "the auth-layer verdict cache must be dropped for the user")
}

// TestUpdateUser_RoleUnchanged_KeepsSessions: edits that do not change the
// role must NOT revoke sessions — an admin fixing a display name must not
// force the user back to login.
func TestUpdateUser_RoleUnchanged_KeepsSessions(t *testing.T) {
	cases := []struct {
		name string
		req  dto.UpdateUserRequest
	}{
		{"role absent", dto.UpdateUserRequest{}},
		{"role set to the same value", dto.UpdateUserRequest{Role: new("quanly")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRoleRepo{user: &domain.User{ID: 8, Username: "ann", Fullname: "An", Role: domain.RoleQuanly}}
			svc := NewUserService(repo, nil, &fakeRoleEventBus{}, "test-secret", "test-salt", nil)

			var notified []uint
			svc.OnSessionsInvalidated = func(_ context.Context, userID uint) {
				notified = append(notified, userID)
			}

			_, err := svc.UpdateUser(context.Background(), 8, tc.req)
			require.NoError(t, err)

			assert.Equal(t, 1, repo.fullRowSaves, "an unchanged role takes the plain update path")
			assert.Equal(t, 0, repo.combinedSaves, "sessions must survive a non-role edit")
			assert.Empty(t, notified, "no cache drop without a session invalidation")
		})
	}
}
