package timesheet

import (
	"api-server/internal/domain"
)

type Workflow struct {
	TimesheetRepo domain.TimesheetRepository
}
