package notification

import (
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/domain"
	"api-server/internal/pkg/utils"
)

// EmployeeNotifier sends push notifications to employees about status changes.
type EmployeeNotifier interface {
	NotifyTimesheetPaid(ctx context.Context, timesheets []*domain.Timesheet)
}

// EmployeeNotificationService sends push notifications to employees about status changes.
type EmployeeNotificationService struct {
	notificationService *NotificationService
	employeeRepo        domain.EmployeeRepository
	logger              *slog.Logger
}

// NewEmployeeNotificationService creates a new employee notification service.
func NewEmployeeNotificationService(
	notificationService *NotificationService,
	employeeRepo domain.EmployeeRepository,
	logger *slog.Logger,
) *EmployeeNotificationService {
	return &EmployeeNotificationService{
		notificationService: notificationService,
		employeeRepo:        employeeRepo,
		logger:              logger,
	}
}

// NotifyTimesheetPaid sends a notification to each employee whose timesheets were paid.
// It groups timesheets by employee and aggregates the total paid amount.
func (s *EmployeeNotificationService) NotifyTimesheetPaid(ctx context.Context, timesheets []*domain.Timesheet) {
	type empTotal struct {
		amount int64
		userID *uint
	}
	byEmployee := make(map[uint]*empTotal)

	for _, ts := range timesheets {
		e, ok := byEmployee[ts.EmployeeID]
		if !ok {
			e = &empTotal{}
			byEmployee[ts.EmployeeID] = e
		}
		e.amount += ts.Amount
	}

	for employeeID, e := range byEmployee {
		emp, err := s.employeeRepo.GetByID(ctx, employeeID)
		if err != nil {
			s.logger.Error("failed to get employee for timesheet paid notification", "employee_id", employeeID, "error", err)
			continue
		}
		e.userID = emp.UserID

		if e.userID == nil {
			continue
		}

		title := "Tiền công đã được thanh toán"
		message := fmt.Sprintf("Bạn đã nhận được %s tiền công.", utils.FormatVND(e.amount))

		if err := s.notificationService.CreateNotification(ctx, *e.userID, domain.NotificationTypeTimesheetPaid, title, message); err != nil {
			s.logger.Error("failed to send timesheet paid notification", "employee_id", employeeID, "error", err)
		}
	}
}
