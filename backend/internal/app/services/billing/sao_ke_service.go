package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"api-server/internal/app/dto"
	appconfig "api-server/internal/app/services/config"
	"api-server/internal/app/services/notification"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// SaoKeService builds the per-Kỳ customer-payment statement (sao kê, PRD
// §5.8/§6.2): the Excel export, the statement email with the workbook
// attached, and the persisted recipient list. A statement always covers one
// Kỳ payment window — the days between the previous Kỳ's pay date (exclusive)
// and this Kỳ's pay date (inclusive).
type SaoKeService struct {
	timesheets domain.TimesheetRepository
	employees  domain.EmployeeRepository
	projects   domain.ProjectRepository
	settings   *appconfig.SettingsService
	email      *notification.EmailService
	logger     *slog.Logger
}

func NewSaoKeService(
	timesheets domain.TimesheetRepository,
	employees domain.EmployeeRepository,
	projects domain.ProjectRepository,
	settings *appconfig.SettingsService,
	email *notification.EmailService,
	logger *slog.Logger,
) *SaoKeService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SaoKeService{
		timesheets: timesheets,
		employees:  employees,
		projects:   projects,
		settings:   settings,
		email:      email,
		logger:     logger,
	}
}

// SaoKeCycle is the Kỳ a report date settles: the Kỳ whose pay date equals
// the date, or — when no pay date matches exactly — the Kỳ whose payment
// window contains it. WorkMonth is the Kỳ's work month ("YYYY-MM"); From/To
// bound the payment window [previous Kỳ's pay date + 1 day, this Kỳ's pay
// date end-of-day] in the canonical timezone.
type SaoKeCycle struct {
	Ky        int       `json:"ky"`
	WorkMonth string    `json:"work_month"`
	PayDate   time.Time `json:"pay_date"`
	From      time.Time `json:"-"`
	To        time.Time `json:"-"`
}

// SaoKeRow is one customer-paid timesheet rendered on the statement.
type SaoKeRow struct {
	TimesheetID  uint
	EmployeeID   uint
	EmployeeName string
	CCCD         string
	ProjectID    uint
	ProjectName  string
	Ky           int
	Amount       int64
	PaymentDate  string // dd/mm/yyyy, empty when only paid_at carried the date
}

// SaoKeProjectGroup is one project's slice of the statement: its rows plus
// the derived employee count and total.
type SaoKeProjectGroup struct {
	ProjectID     uint
	ProjectName   string
	EmployeeCount int
	Total         int64
	Rows          []SaoKeRow
}

// SaoKeData is the full statement payload for one Kỳ payment window.
type SaoKeData struct {
	Cycle        SaoKeCycle
	ReportDate   time.Time
	Rows         []SaoKeRow
	Groups       []SaoKeProjectGroup
	Total        int64
	TimesheetIDs []uint
}

// saoKeCycleCandidates lists the Kỳ buckets whose payment window can touch a
// date: Kỳ 1–4 of the date's month plus Kỳ 4 of the previous month (it pays
// on day 1 of the date's month).
func saoKeCycleCandidates(d time.Time) []SaoKeCycle {
	firstOfMonth := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, clock.DefaultLocation)
	prevMonth := firstOfMonth.AddDate(0, -1, 0)
	out := make([]SaoKeCycle, 0, clock.TimesheetKyCount+1)
	for ky := 1; ky <= clock.TimesheetKyCount; ky++ {
		out = append(out, buildSaoKeCycle(ky, firstOfMonth))
	}
	out = append(out, buildSaoKeCycle(clock.TimesheetKyCount, prevMonth))
	return out
}

// buildSaoKeCycle assembles one Kỳ's payment window: the canonical pay date
// plus the day after the previous Kỳ's pay date.
func buildSaoKeCycle(ky int, workMonth time.Time) SaoKeCycle {
	payDate := clock.PayDate(ky, workMonth.Year(), workMonth.Month())
	prevKy, prevMonth := ky-1, workMonth
	if prevKy < 1 {
		prevKy = clock.TimesheetKyCount
		prevMonth = workMonth.AddDate(0, -1, 0)
	}
	prevPay := clock.PayDate(prevKy, prevMonth.Year(), prevMonth.Month())
	from := time.Date(prevPay.Year(), prevPay.Month(), prevPay.Day(), 0, 0, 0, 0, clock.DefaultLocation).AddDate(0, 0, 1)
	to := time.Date(payDate.Year(), payDate.Month(), payDate.Day(), 23, 59, 59, 999999999, clock.DefaultLocation)
	return SaoKeCycle{
		Ky:        ky,
		WorkMonth: workMonth.Format("2006-01"),
		PayDate:   payDate,
		From:      from,
		To:        to,
	}
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.In(clock.DefaultLocation).Date()
	by, bm, bd := b.In(clock.DefaultLocation).Date()
	return ay == by && am == bm && ad == bd
}

// ResolveSaoKeCycle maps a report date onto the Kỳ it settles. Exact pay-date
// match wins; otherwise the Kỳ whose payment window contains the date does.
// The windows partition the calendar, so a match always exists.
func ResolveSaoKeCycle(reportDate time.Time) SaoKeCycle {
	d := time.Date(reportDate.Year(), reportDate.Month(), reportDate.Day(), 0, 0, 0, 0, clock.DefaultLocation)
	candidates := saoKeCycleCandidates(d)
	for _, c := range candidates {
		if sameDate(c.PayDate, d) {
			return c
		}
	}
	for _, c := range candidates {
		if !d.Before(c.From) && !d.After(c.To) {
			return c
		}
	}
	// Defensive: the windows above cover every calendar day, so this is
	// unreachable — fall back to the Kỳ whose pay date is the latest at or
	// before the date.
	fallback := candidates[0]
	for _, c := range candidates {
		if !c.PayDate.After(d) {
			fallback = c
		}
	}
	return fallback
}

// ParseSaoKeReportDate parses the YYYY-MM-DD report_date request parameter,
// defaulting to today in the canonical timezone when empty.
func ParseSaoKeReportDate(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return clock.Now(), nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", trimmed, clock.DefaultLocation)
	if err != nil {
		return time.Time{}, fmt.Errorf("report_date phải có dạng YYYY-MM-DD")
	}
	return parsed, nil
}

// WorkWindow returns the Kỳ's canonical work-day window (inclusive, HCM
// timezone, date-only) — the "từ–đến" range rendered next to the Kỳ label.
func (c SaoKeCycle) WorkWindow() (time.Time, time.Time) {
	month, err := time.ParseInLocation("2006-01", c.WorkMonth, clock.DefaultLocation)
	if err != nil {
		return c.PayDate, c.PayDate
	}
	pc := clock.TimesheetPayCycle{Ky: c.Ky, WorkMonth: month}
	return pc.CycleWindow()
}

// KyLabel renders the Kỳ work range the way the workbook and email body
// repeat it, e.g. "Kỳ 1 (01/11/2026 – 07/11/2026)".
func (c SaoKeCycle) KyLabel() string {
	from, to := c.WorkWindow()
	return fmt.Sprintf("Kỳ %d (%s – %s)", c.Ky, from.Format("02/01/2006"), to.Format("02/01/2006"))
}

// Collect loads the customer-paid timesheets of the resolved Kỳ payment
// window and groups them per project for the statement sheets.
func (s *SaoKeService) Collect(ctx context.Context, reportDate time.Time) (*SaoKeData, error) {
	cycle := ResolveSaoKeCycle(reportDate)

	var paid []*domain.Timesheet
	if err := s.timesheets.GetPaidTimesheetsInDateRange(ctx, cycle.From, cycle.To, &paid); err != nil {
		return nil, fmt.Errorf("sao ke: load paid timesheets: %w", err)
	}

	data := &SaoKeData{Cycle: cycle, ReportDate: reportDate}
	if len(paid) == 0 {
		return data, nil
	}

	// Employee identities (name, CCCD) for the per-project sheets.
	employeeIDs := make([]int64, 0, len(paid))
	seenEmployees := make(map[uint]struct{}, len(paid))
	for _, ts := range paid {
		if _, ok := seenEmployees[ts.EmployeeID]; ok {
			continue
		}
		seenEmployees[ts.EmployeeID] = struct{}{}
		employeeIDs = append(employeeIDs, int64(ts.EmployeeID))
	}
	names := make(map[uint]string, len(employeeIDs))
	cccds := make(map[uint]string, len(employeeIDs))
	if len(employeeIDs) > 0 {
		employees, err := s.employees.GetByIDs(ctx, employeeIDs)
		if err != nil {
			return nil, fmt.Errorf("sao ke: load employees: %w", err)
		}
		for _, e := range employees {
			names[e.ID] = e.Fullname
			cccds[e.ID] = e.CCCD
		}
	}

	// Real project names — the per-project sheets carry them as titles.
	projectIDs := make([]uint, 0, 8)
	seenProjects := make(map[uint]struct{}, 8)
	for _, ts := range paid {
		if _, ok := seenProjects[ts.ProjectID]; ok {
			continue
		}
		seenProjects[ts.ProjectID] = struct{}{}
		projectIDs = append(projectIDs, ts.ProjectID)
	}
	projectNames := make(map[uint]string, len(projectIDs))
	if len(projectIDs) > 0 {
		projects, err := s.projects.GetByIDs(ctx, projectIDs)
		if err != nil {
			return nil, fmt.Errorf("sao ke: load projects: %w", err)
		}
		for id, p := range projects {
			if p == nil {
				continue
			}
			switch {
			case p.Name != "":
				projectNames[id] = p.Name
			case p.Code != "":
				projectNames[id] = p.Code
			}
		}
	}

	data.Rows = make([]SaoKeRow, 0, len(paid))
	groups := make(map[uint]*SaoKeProjectGroup, len(projectIDs))
	for _, ts := range paid {
		paymentDate := ""
		if ts.PaymentDate != nil && !ts.PaymentDate.IsZero() {
			paymentDate = ts.PaymentDate.Format("02/01/2006")
		}
		projectName := projectNames[ts.ProjectID]
		if projectName == "" {
			projectName = "(dự án đã xoá)"
		}
		row := SaoKeRow{
			TimesheetID:  ts.ID,
			EmployeeID:   ts.EmployeeID,
			EmployeeName: names[ts.EmployeeID],
			CCCD:         cccds[ts.EmployeeID],
			ProjectID:    ts.ProjectID,
			ProjectName:  projectName,
			Ky:           cycle.Ky,
			Amount:       ts.PaidAmount,
			PaymentDate:  paymentDate,
		}
		if row.Amount == 0 {
			row.Amount = ts.Amount
		}
		data.Rows = append(data.Rows, row)
		data.TimesheetIDs = append(data.TimesheetIDs, ts.ID)
		data.Total += row.Amount

		group, ok := groups[ts.ProjectID]
		if !ok {
			group = &SaoKeProjectGroup{ProjectID: ts.ProjectID, ProjectName: projectName}
			groups[ts.ProjectID] = group
		}
		group.Rows = append(group.Rows, row)
		group.Total += row.Amount
	}

	// Deterministic order: projects by id, rows by employee then timesheet.
	for _, id := range projectIDs {
		if group, ok := groups[id]; ok {
			sortSaoKeRows(group.Rows)
			distinct := make(map[uint]struct{}, len(group.Rows))
			for _, row := range group.Rows {
				distinct[row.EmployeeID] = struct{}{}
			}
			group.EmployeeCount = len(distinct)
			data.Groups = append(data.Groups, *group)
		}
	}
	sortSaoKeRows(data.Rows)
	return data, nil
}

func sortSaoKeRows(rows []SaoKeRow) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0; j-- {
			a, b := rows[j-1], rows[j]
			if a.EmployeeName < b.EmployeeName ||
				(a.EmployeeName == b.EmployeeName && a.TimesheetID < b.TimesheetID) {
				break
			}
			rows[j-1], rows[j] = rows[j], rows[j-1]
		}
	}
}

// SaoKeRecipients is the persisted statement-mailing list: the To addresses
// (used when sending) and the optional CC list.
type SaoKeRecipients struct {
	Recipients []string `json:"recipients"`
	CC         []string `json:"cc"`
}

// GetRecipients reads the persisted statement recipients. Missing rows read
// as empty lists — recipients are optional until the first save.
func (s *SaoKeService) GetRecipients(ctx context.Context) (*SaoKeRecipients, error) {
	recipients, err := s.readRecipientsKey(ctx, appconfig.SettingKeySaoKeEmailRecipients)
	if err != nil {
		return nil, err
	}
	cc, err := s.readRecipientsKey(ctx, appconfig.SettingKeySaoKeEmailCC)
	if err != nil {
		return nil, err
	}
	return &SaoKeRecipients{Recipients: recipients, CC: cc}, nil
}

func (s *SaoKeService) readRecipientsKey(ctx context.Context, key string) ([]string, error) {
	setting, err := s.settings.GetSettingByKeyAuthoritative(ctx, key)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("sao ke: read recipients setting %s: %w", key, err)
	}
	if setting.Value == nil || strings.TrimSpace(*setting.Value) == "" {
		return []string{}, nil
	}
	var addresses []string
	if err := json.Unmarshal([]byte(*setting.Value), &addresses); err != nil {
		s.logger.Warn("sao ke: unparsable recipients setting", "key", key, "error", err)
		return []string{}, nil
	}
	return addresses, nil
}

// SaveRecipients validates and persists the statement mailing list. Both
// lists may be empty (a saved-but-unused list is a legitimate state); every
// supplied entry must be a syntactic email address.
func (s *SaoKeService) SaveRecipients(ctx context.Context, recipients *SaoKeRecipients, updatedBy uint) error {
	to := normalizeEmailList(recipients.Recipients)
	cc := normalizeEmailList(recipients.CC)
	for _, address := range append(append([]string{}, to...), cc...) {
		if _, err := domain.ParseEmailAddress(address); err != nil {
			return domain.NewValidationError(fmt.Sprintf("địa chỉ email không hợp lệ: %s", address))
		}
	}
	if err := s.saveRecipientsKey(ctx, appconfig.SettingKeySaoKeEmailRecipients, to, updatedBy); err != nil {
		return err
	}
	return s.saveRecipientsKey(ctx, appconfig.SettingKeySaoKeEmailCC, cc, updatedBy)
}

func normalizeEmailList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func (s *SaoKeService) saveRecipientsKey(ctx context.Context, key string, addresses []string, updatedBy uint) error {
	encoded, err := json.Marshal(addresses)
	if err != nil {
		return fmt.Errorf("sao ke: encode recipients %s: %w", key, err)
	}
	value := string(encoded)
	existing, err := s.settings.GetSettingByKeyAuthoritative(ctx, key)
	if err != nil {
		if domain.IsNotFoundError(err) {
			_, createErr := s.settings.CreateSetting(ctx, &domain.Settings{
				Key:       key,
				Value:     &value,
				ValueType: domain.ValueTypeJSON,
			}, updatedBy)
			if createErr != nil {
				return fmt.Errorf("sao ke: create recipients setting %s: %w", key, createErr)
			}
			return nil
		}
		return fmt.Errorf("sao ke: load recipients setting %s: %w", key, err)
	}
	if _, err := s.settings.UpdateSetting(ctx, existing.ID, map[string]any{
		"value":      value,
		"value_type": string(domain.ValueTypeJSON),
	}, updatedBy); err != nil {
		return fmt.Errorf("sao ke: update recipients setting %s: %w", key, err)
	}
	return nil
}

// SendEmail builds the statement workbook for the resolved Kỳ and mails it to
// the request's recipients. The body is only the transaction-detail table
// plus a pointer to the attached workbook.
func (s *SaoKeService) SendEmail(ctx context.Context, reportDate time.Time, recipients, cc []string) (string, error) {
	to := normalizeEmailList(recipients)
	if len(to) == 0 {
		return "", domain.NewValidationError("cần ít nhất một địa chỉ người nhận (To)")
	}
	cc = normalizeEmailList(cc)

	data, err := s.Collect(ctx, reportDate)
	if err != nil {
		return "", err
	}
	workbook, filename, err := renderSaoKeWorkbook(data)
	if err != nil {
		return "", err
	}

	payload := &dto.SendEmailRequest{
		Recipients: to,
		Cc:         cc,
		Subject:    fmt.Sprintf("Sao kê thanh toán lương – %s", data.ReportDate.Format("02/01/2006")),
		HTMLBody:   renderSaoKeEmailBody(data),
		Attachments: []dto.AttachmentData{{
			Filename:    filename,
			ContentType: saoKeContentType,
			Content:     workbook,
		}},
	}
	return s.email.SendGenericEmail(ctx, payload)
}

// Export renders the statement workbook bytes and its download filename
// sao_ke_tt_<DD-MM-YYYY>.xlsx. The returned data carries the row count for
// the export audit event.
func (s *SaoKeService) Export(ctx context.Context, reportDate time.Time) (*SaoKeData, []byte, string, error) {
	data, err := s.Collect(ctx, reportDate)
	if err != nil {
		return nil, nil, "", err
	}
	workbook, filename, err := renderSaoKeWorkbook(data)
	if err != nil {
		return nil, nil, "", err
	}
	return data, workbook, filename, nil
}
