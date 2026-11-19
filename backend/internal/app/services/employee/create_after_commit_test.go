package employee

import (
	"context"
	"testing"
	"time"

	userservice "api-server/internal/app/services/user"
	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// Shared stubs for the employee service tests: an in-memory employee repo, a
// pass-through transaction manager, a minimal user repo and a no-op event bus.

// stubEmployeeRepo captures Create/Update calls and serves GetByID from an
// in-memory map. Embeds the interface so we only override what we use.
type stubEmployeeRepo struct {
	domain.EmployeeRepository
	byID        map[uint]*domain.Employee
	lockedByID  map[uint]*domain.Employee
	getErr      error
	updatedID   uint
	cols        map[string]any
	updateErr   error
	updateCalls int
	columnCalls int
	lockReads   int
	created     []*domain.Employee
}

func (s *stubEmployeeRepo) Create(_ context.Context, employee *domain.Employee) error {
	employee.ID = uint(len(s.created) + 1)
	s.created = append(s.created, employee)
	return nil
}

func (s *stubEmployeeRepo) GetByCCCD(_ context.Context, _ string) (*domain.Employee, error) {
	return nil, domain.NewNotFoundError("employee not found")
}

func (s *stubEmployeeRepo) GetByID(_ context.Context, id uint) (*domain.Employee, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.byID[id], nil
}

func (s *stubEmployeeRepo) GetByIDForUpdate(_ context.Context, id uint) (*domain.Employee, error) {
	s.lockReads++
	if s.getErr != nil {
		return nil, s.getErr
	}
	if employee, ok := s.lockedByID[id]; ok {
		return employee, nil
	}
	return s.byID[id], nil
}

func (s *stubEmployeeRepo) UpdateColumns(_ context.Context, id uint, cols map[string]any) error {
	s.columnCalls++
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updatedID = id
	s.cols = cols
	return nil
}

func (s *stubEmployeeRepo) Update(_ context.Context, _ *domain.Employee) error {
	s.updateCalls++
	return s.updateErr
}

type directTransactionManager struct{}

func (directTransactionManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (directTransactionManager) WithTransactionResult(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	return fn(ctx)
}

type createEmployeeTestUserRepo struct {
	domain.UserRepository
	users     []*domain.User
	createErr error
}

func (r *createEmployeeTestUserRepo) GetUsernamesByPrefix(context.Context, string) ([]string, error) {
	return nil, nil
}

func (r *createEmployeeTestUserRepo) GetByUsername(context.Context, string) (*domain.User, error) {
	return nil, domain.NewNotFoundError("user not found")
}

func (r *createEmployeeTestUserRepo) GetByID(_ context.Context, id uint) (*domain.User, error) {
	for _, stored := range r.users {
		if stored.ID == id {
			return stored, nil
		}
	}
	return nil, domain.NewNotFoundError("user not found")
}

func (r *createEmployeeTestUserRepo) Create(_ context.Context, user *domain.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	user.ID = uint(len(r.users) + 1)
	r.users = append(r.users, user)
	return nil
}

type createEmployeeTestEventBus struct{}

func (createEmployeeTestEventBus) Publish(context.Context, ...domain.DomainEvent) error {
	return nil
}

func (createEmployeeTestEventBus) Subscribe(string, domain.EventHandler) {}
func (createEmployeeTestEventBus) SubscribeAll(domain.EventHandler)      {}

type employeeCreationEventBus struct {
	createEmployeeTestEventBus
	created chan context.Context
}

func (b employeeCreationEventBus) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	for _, event := range events {
		if event.EventType() == "EmployeeCreated" {
			b.created <- ctx
		}
	}
	return nil
}

func TestCreateEmployeePublishesOnlyAfterOuterCommit(t *testing.T) {
	for _, commit := range []bool{false, true} {
		name := "rollback"
		if commit {
			name = "commit"
		}
		t.Run(name, func(t *testing.T) {
			events := employeeCreationEventBus{created: make(chan context.Context, 1)}
			users := &createEmployeeTestUserRepo{}
			service := &EmployeeService{
				EmployeeRepo: &stubEmployeeRepo{}, UserRepo: users,
				UserService:        userservice.NewUserService(users, nil, events, "test-secret", "test-salt", nil),
				TransactionManager: directTransactionManager{}, events: events,
			}
			transaction := &domain.TransactionContext{IsTransactional: true}
			ctx, cancel := context.WithCancel(domain.WithTransactionContext(context.Background(), transaction))
			_, err := service.CreateEmployeeFromImport(ctx, &domain.Employee{
				Fullname: "Synthetic Import Employee", CCCD: "099260900502",
			}, 99)
			require.NoError(t, err)
			cancel()
			select {
			case <-events.created:
				t.Fatal("employee event published before outer transaction committed")
			default:
			}
			if !commit {
				return
			}
			transaction.RunAfterCommitCallbacks()
			select {
			case eventCtx := <-events.created:
				_, transactional := domain.GetTransactionFromContext(eventCtx)
				require.False(t, transactional, "consumer must not reuse the committed transaction")
				require.NoError(t, eventCtx.Err(), "consumer must outlive the import context")
			case <-time.After(time.Second):
				t.Fatal("employee event missing after commit")
			}
		})
	}
}
