package dashboard

import (
	"api-server/internal/pkg/clock"
	"context"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/pkg/timeutil"
)

func derefStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
func (s *Service) GetTopPaidEmployees(ctx context.Context, req *dto.TopPaidEmployeesRequest) (*dto.TopPaidEmployeesResponse, error) {
	s.logger.Info("Getting top paid employees", "month", req.Month, "limit", req.Limit)

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	var startDate, endDate *time.Time
	period := "all_time"

	if req.Month != "" {
		t, err := time.Parse("2006-01", req.Month)
		if err == nil {
			start := timeutil.StartOfMonth(t)
			end := timeutil.EndOfMonth(t).Add(time.Nanosecond)
			startDate = &start
			endDate = &end
			period = req.Month
		}
	}

	rows, err := s.TimesheetAnalyticsRepo.GetTopPaidEmployees(ctx, startDate, endDate, limit)
	if err != nil {
		s.logger.Error("Failed to get top paid employees", "error", err)
		return nil, err
	}

	cutoff := clock.Now().AddDate(0, 0, -14)
	items := make([]dto.TopPaidEmployeeItem, 0, len(rows))
	for i, row := range rows {
		isActive := false
		if row.LastTimesheetDate != nil {
			if lastDate, err := time.Parse("2006-01-02", *row.LastTimesheetDate); err == nil {
				isActive = lastDate.After(cutoff) || lastDate.Equal(cutoff)
			}
		}
		items = append(items, dto.TopPaidEmployeeItem{
			Rank:              i + 1,
			EmployeeID:        row.EmployeeID,
			EmployeeName:      row.EmployeeName,
			TotalPaidVND:      int64(row.TotalPaid),
			IsActive:          isActive,
			LastTimesheetDate: derefStr(row.LastTimesheetDate),
		})
	}

	return &dto.TopPaidEmployeesResponse{
		Employees: items,
		Period:    period,
	}, nil
}
