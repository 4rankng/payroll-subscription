package billing

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
	"unicode"

	"api-server/internal/app/services/excel"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"golang.org/x/text/unicode/norm"
)

// PaymentHistoryService serves the read-only payment history (Lịch sử trả
// lương): one row per completed bank entry of a work month — a posting = the
// set of customer-paid timesheets one transfer-result row marked paid for one
// employee in one Kỳ. Split transfers (same employee+Kỳ, different bank
// references) stay visible as separate rows. Read-only: nothing here mutates
// the books.
type PaymentHistoryService struct {
	timesheets       domain.TimesheetRepository
	employees        domain.EmployeeRepository
	projects         domain.ProjectRepository
	kyPaid           domain.EmployeePayCycleRepository
	uploads          domain.TransferUploadRepository
	projectEmployees domain.ProjectEmployeeRepository
	logger           *slog.Logger
}

func NewPaymentHistoryService(
	timesheets domain.TimesheetRepository,
	employees domain.EmployeeRepository,
	projects domain.ProjectRepository,
	kyPaid domain.EmployeePayCycleRepository,
	uploads domain.TransferUploadRepository,
	projectEmployees domain.ProjectEmployeeRepository,
	logger *slog.Logger,
) *PaymentHistoryService {
	if logger == nil {
		logger = slog.Default()
	}
	return &PaymentHistoryService{
		timesheets:       timesheets,
		employees:        employees,
		projects:         projects,
		kyPaid:           kyPaid,
		uploads:          uploads,
		projectEmployees: projectEmployees,
		logger:           logger,
	}
}

// PaymentHistoryQuery is the filter set of the payment-history read. Month is
// the work month (YYYY-MM, empty = current month); Ky 0 = all Kỳ.
type PaymentHistoryQuery struct {
	Month    string
	Ky       int
	Search   string
	Page     int
	PageSize int
}

// PaymentHistoryRow is one completed bank entry — the flat ledger line of the
// payment history.
type PaymentHistoryRow struct {
	Stt                 int    `json:"stt"`
	EmployeeName        string `json:"employee_name"`
	CCCD                string `json:"cccd"`
	ProjectName         string `json:"project_name"`
	Ky                  int    `json:"ky"`
	KyWindow            string `json:"ky_window"`
	PayDate             string `json:"pay_date"`
	TransferNote        string `json:"transfer_note"`
	BankTransactionCode string `json:"bank_transaction_code"`
	ProcessedAt         string `json:"processed_at"`
	AmountVnd           int64  `json:"amount_vnd"`
	ActualReceivedVnd   int64  `json:"actual_received_vnd"`
}

// PaymentHistorySummary aggregates the whole filtered set (not just the page).
type PaymentHistorySummary struct {
	TotalAmountVnd int64 `json:"total_amount_vnd"`
	BankEntryCount int   `json:"bank_entry_count"`
	EmployeeCount  int   `json:"employee_count"`
}

// PaymentHistoryResult is the paginated payment-history response.
type PaymentHistoryResult struct {
	Summary  PaymentHistorySummary `json:"summary"`
	Rows     []PaymentHistoryRow   `json:"rows"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

// PaymentHistoryExport is the rendered workbook plus the record count the
// export audit event needs.
type PaymentHistoryExport struct {
	Workbook     []byte
	RecordCount  int
	ProjectCount int
}

// paymentHistoryEntry is one grouped bank entry before row projection.
type paymentHistoryEntry struct {
	employeeID   uint
	employeeName string
	cccd         string
	projectIDs   []uint
	projectNames []string
	byProjectVnd map[uint]int64
	ky           int
	windowFrom   time.Time
	windowTo     time.Time
	payDate      time.Time
	hasPayDate   bool
	transferNote string
	bankCode     string
	processedAt  time.Time
	hasProcessed bool
	amountVnd    int64
	actualVnd    int64
}

type employeeCycleKey struct {
	employeeID uint
	cycle      int
}

type projectEmployeeKey struct {
	projectID  uint
	employeeID uint
}

type entryKey struct {
	employeeID uint
	ky         int
	ref        string
	paidAt     string
	payDate    string
}

// ListPaymentHistory returns one page of bank entries for the filtered work
// month plus whole-set summary totals.
func (s *PaymentHistoryService) ListPaymentHistory(ctx context.Context, q PaymentHistoryQuery) (*PaymentHistoryResult, error) {
	entries, err := s.buildEntries(ctx, q)
	if err != nil {
		return nil, err
	}

	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 500 {
		pageSize = 500
	}

	summary := PaymentHistorySummary{BankEntryCount: len(entries)}
	employees := make(map[uint]struct{}, len(entries))
	for _, e := range entries {
		summary.TotalAmountVnd += e.amountVnd
		employees[e.employeeID] = struct{}{}
	}
	summary.EmployeeCount = len(employees)

	start := (page - 1) * pageSize
	if start > len(entries) {
		start = len(entries)
	}
	end := start + pageSize
	if end > len(entries) {
		end = len(entries)
	}

	rows := make([]PaymentHistoryRow, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, projectPaymentHistoryRow(entries[i], i+1))
	}

	return &PaymentHistoryResult{
		Summary:  summary,
		Rows:     rows,
		Total:    len(entries),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ExportPaymentHistory renders the per-project Excel workbook of the filtered
// work month: one sheet per project — STT | Nhân viên | CCCD | Vị trí | Số
// tiền | Ngày thanh toán.
func (s *PaymentHistoryService) ExportPaymentHistory(ctx context.Context, q PaymentHistoryQuery) (*PaymentHistoryExport, error) {
	entries, err := s.buildEntries(ctx, q)
	if err != nil {
		return nil, err
	}

	positions, err := s.loadPositions(ctx, entries)
	if err != nil {
		return nil, err
	}

	byProject := make(map[uint][]excel.PaymentHistoryExportRow)
	nameByProject := make(map[uint]string, 8)
	projectIDs := make([]uint, 0, 8)
	for _, e := range entries {
		for i, pid := range e.projectIDs {
			if _, seen := byProject[pid]; !seen {
				projectIDs = append(projectIDs, pid)
				if i < len(e.projectNames) {
					nameByProject[pid] = e.projectNames[i]
				}
			}
			byProject[pid] = append(byProject[pid], excel.PaymentHistoryExportRow{
				EmployeeName: e.employeeName,
				CCCD:         e.cccd,
				Position:     positions[projectEmployeeKey{projectID: pid, employeeID: e.employeeID}],
				AmountVnd:    e.byProjectVnd[pid],
				PayDate:      payDateDisplay(e),
			})
		}
	}
	sort.Slice(projectIDs, func(i, j int) bool { return projectIDs[i] < projectIDs[j] })

	sheets := make([]excel.PaymentHistoryProjectSheet, 0, len(projectIDs))
	for _, pid := range projectIDs {
		rows := byProject[pid]
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].EmployeeName != rows[j].EmployeeName {
				return rows[i].EmployeeName < rows[j].EmployeeName
			}
			return rows[i].PayDate < rows[j].PayDate
		})
		sheets = append(sheets, excel.PaymentHistoryProjectSheet{
			ProjectID:   pid,
			ProjectName: nameByProject[pid],
			Rows:        rows,
		})
	}

	workbook, err := excel.RenderPaymentHistoryWorkbook(sheets)
	if err != nil {
		return nil, fmt.Errorf("payment history export: %w", err)
	}

	count := 0
	for _, sheet := range sheets {
		count += len(sheet.Rows)
	}
	return &PaymentHistoryExport{Workbook: workbook, RecordCount: count, ProjectCount: len(sheets)}, nil
}

// buildEntries loads the work month's customer-paid timesheets, groups them
// into bank entries (employee × Kỳ × transfer posting), joins employee names,
// project names and the transfer upload record, then applies the Kỳ and
// search filters and sorts the result.
func (s *PaymentHistoryService) buildEntries(ctx context.Context, q PaymentHistoryQuery) ([]paymentHistoryEntry, error) {
	monthTime, err := resolveHistoryMonth(q.Month)
	if err != nil {
		return nil, err
	}
	month := monthTime.Format("2006-01")
	monthStart := time.Date(monthTime.Year(), monthTime.Month(), 1, 0, 0, 0, 0, clock.DefaultLocation)
	monthEnd := monthStart.AddDate(0, 1, -1) // ApplyDateRange treats ToDate as inclusive

	paid, err := s.timesheets.List(ctx, domain.TimesheetFilters{
		PaymentStatus: []domain.PaymentStatus{domain.PaymentStatusPaid},
		FromDate:      &monthStart,
		ToDate:        &monthEnd,
		SkipRelations: true,
	})
	if err != nil {
		return nil, fmt.Errorf("payment history: load paid timesheets: %w", err)
	}

	order := make([]entryKey, 0, len(paid))
	groups := make(map[entryKey]*paymentHistoryEntry, len(paid))
	for _, ts := range paid {
		ky := clock.KyFromWorkDay(ts.Date.Day())
		var ref string
		if ts.PaymentReference != nil {
			ref = strings.TrimSpace(*ts.PaymentReference)
		}
		key := entryKey{employeeID: ts.EmployeeID, ky: ky, ref: ref}
		if ts.PaidAt != nil {
			key.paidAt = ts.PaidAt.In(clock.DefaultLocation).Format(time.RFC3339Nano)
		}
		if ts.PaymentDate != nil {
			key.payDate = ts.PaymentDate.Format("2006-01-02")
		}
		entry := groups[key]
		if entry == nil {
			from, to := (clock.TimesheetPayCycle{Ky: ky, WorkMonth: monthStart}).CycleWindow()
			entry = &paymentHistoryEntry{
				employeeID:   ts.EmployeeID,
				projectIDs:   make([]uint, 0, 2),
				byProjectVnd: make(map[uint]int64, 2),
				ky:           ky,
				windowFrom:   from,
				windowTo:     to,
				bankCode:     ref,
			}
			groups[key] = entry
			order = append(order, key)
		}
		entry.amountVnd += ts.Amount
		entry.actualVnd += ts.PaidAmount
		if _, seen := entry.byProjectVnd[ts.ProjectID]; !seen {
			entry.projectIDs = append(entry.projectIDs, ts.ProjectID)
		}
		entry.byProjectVnd[ts.ProjectID] += ts.PaidAmount
		if ts.PaymentDate != nil && (!entry.hasPayDate || ts.PaymentDate.After(entry.payDate)) {
			entry.payDate = *ts.PaymentDate
			entry.hasPayDate = true
		}
		if ts.PaidAt != nil && (!entry.hasProcessed || ts.PaidAt.After(entry.processedAt)) {
			entry.processedAt = *ts.PaidAt
			entry.hasProcessed = true
		}
	}

	entries := make([]paymentHistoryEntry, 0, len(order))
	for _, key := range order {
		entries = append(entries, *groups[key])
	}

	if err := s.resolveLookups(ctx, month, entries); err != nil {
		return nil, err
	}

	entries = filterHistoryEntries(entries, q.Ky, q.Search)

	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.ky != b.ky {
			return a.ky < b.ky
		}
		if a.employeeName != b.employeeName {
			return a.employeeName < b.employeeName
		}
		if a.bankCode != b.bankCode {
			return a.bankCode < b.bankCode
		}
		return a.processedAt.Before(b.processedAt)
	})
	return entries, nil
}

// resolveLookups joins employee names/CCCDs, project names and the transfer
// upload record (transfer note) onto the grouped entries.
func (s *PaymentHistoryService) resolveLookups(ctx context.Context, month string, entries []paymentHistoryEntry) error {
	employeeIDs := make([]int64, 0, len(entries))
	seenEmp := make(map[uint]struct{}, len(entries))
	projectIDs := make([]uint, 0, len(entries))
	seenProj := make(map[uint]struct{}, 8)
	for i := range entries {
		e := &entries[i]
		if _, ok := seenEmp[e.employeeID]; !ok {
			seenEmp[e.employeeID] = struct{}{}
			employeeIDs = append(employeeIDs, int64(e.employeeID))
		}
		for _, pid := range e.projectIDs {
			if _, ok := seenProj[pid]; !ok {
				seenProj[pid] = struct{}{}
				projectIDs = append(projectIDs, pid)
			}
		}
	}

	employees, err := s.employees.GetByIDs(ctx, employeeIDs)
	if err != nil {
		return fmt.Errorf("payment history: load employees: %w", err)
	}
	empByID := make(map[uint]*domain.Employee, len(employees))
	for _, e := range employees {
		empByID[e.ID] = e
	}

	projectNames := s.projectNamesByID(ctx, projectIDs)

	notes, err := s.transferNotesByEmployeeCycle(ctx, month)
	if err != nil {
		return err
	}

	for i := range entries {
		e := &entries[i]
		if emp := empByID[e.employeeID]; emp != nil {
			e.employeeName = emp.Fullname
			e.cccd = emp.CCCD
		}
		e.projectNames = make([]string, 0, len(e.projectIDs))
		for _, pid := range e.projectIDs {
			e.projectNames = append(e.projectNames, projectNames[pid])
		}
		e.transferNote = notes[employeeCycleKey{employeeID: e.employeeID, cycle: e.ky}]
	}
	return nil
}

// transferNotesByEmployeeCycle maps (employee, Kỳ) → the original filename of
// the transfer upload record that verified the payment (Ghi chú CK). The
// schema keeps one upload pointer per employee per Kỳ (employee_pay_cycles.
// source_upload_id), so split postings of the same Kỳ share the note.
func (s *PaymentHistoryService) transferNotesByEmployeeCycle(ctx context.Context, month string) (map[employeeCycleKey]string, error) {
	cycles, err := s.kyPaid.ListByWorkMonth(ctx, month)
	if err != nil {
		return nil, fmt.Errorf("payment history: load employee pay cycles: %w", err)
	}
	uploadIDs := make(map[uint64]struct{}, len(cycles))
	for _, c := range cycles {
		uploadIDs[c.SourceUploadID] = struct{}{}
	}

	uploads, err := s.uploads.ListByWorkMonth(ctx, month, 1000)
	if err != nil {
		return nil, fmt.Errorf("payment history: load transfer uploads: %w", err)
	}
	filenameByUpload := make(map[uint64]string, len(uploads))
	for _, u := range uploads {
		if _, wanted := uploadIDs[u.ID]; wanted {
			filenameByUpload[u.ID] = u.OriginalFilename
		}
	}

	notes := make(map[employeeCycleKey]string, len(cycles))
	for _, c := range cycles {
		notes[employeeCycleKey{employeeID: c.EmployeeID, cycle: c.Cycle}] = filenameByUpload[c.SourceUploadID]
	}
	return notes, nil
}

func (s *PaymentHistoryService) projectNamesByID(ctx context.Context, projectIDs []uint) map[uint]string {
	names := make(map[uint]string, len(projectIDs))
	if len(projectIDs) == 0 {
		return names
	}
	projects, err := s.projects.GetByIDs(ctx, projectIDs)
	if err != nil {
		s.logger.Warn("payment history: load project names failed", "error", err)
		return names
	}
	for id, p := range projects {
		names[id] = p.Name
	}
	return names
}

func (s *PaymentHistoryService) loadPositions(ctx context.Context, entries []paymentHistoryEntry) (map[projectEmployeeKey]string, error) {
	positions := make(map[projectEmployeeKey]string)
	if len(entries) == 0 {
		return positions, nil
	}
	projectIDs := make([]uint, 0, len(entries))
	employeeIDs := make([]uint, 0, len(entries))
	seenProj := make(map[uint]struct{}, len(entries))
	seenEmp := make(map[uint]struct{}, len(entries))
	for _, e := range entries {
		if _, ok := seenEmp[e.employeeID]; !ok {
			seenEmp[e.employeeID] = struct{}{}
			employeeIDs = append(employeeIDs, e.employeeID)
		}
		for _, pid := range e.projectIDs {
			if _, ok := seenProj[pid]; !ok {
				seenProj[pid] = struct{}{}
				projectIDs = append(projectIDs, pid)
			}
		}
	}
	assignments, err := s.projectEmployees.GetActiveAssignmentsByProjectsAndEmployees(ctx, projectIDs, employeeIDs)
	if err != nil {
		return nil, fmt.Errorf("payment history: load positions: %w", err)
	}
	for _, a := range assignments {
		positions[projectEmployeeKey{projectID: a.ProjectID, employeeID: a.EmployeeID}] = a.Position
	}
	return positions, nil
}

// projectPaymentHistoryRow projects one entry onto its API row. stt is the
// caller-assigned 1-based ledger line number.
func projectPaymentHistoryRow(e paymentHistoryEntry, stt int) PaymentHistoryRow {
	row := PaymentHistoryRow{
		Stt:                 stt,
		EmployeeName:        e.employeeName,
		CCCD:                e.cccd,
		ProjectName:         strings.Join(e.projectNames, ", "),
		Ky:                  e.ky,
		KyWindow:            fmt.Sprintf("%s → %s", e.windowFrom.Format("02/01"), e.windowTo.Format("02/01")),
		TransferNote:        e.transferNote,
		BankTransactionCode: e.bankCode,
		AmountVnd:           e.amountVnd,
		ActualReceivedVnd:   e.actualVnd,
	}
	if e.hasPayDate {
		row.PayDate = e.payDate.Format("2006-01-02")
	}
	if e.hasProcessed {
		row.ProcessedAt = e.processedAt.In(clock.DefaultLocation).Format(time.RFC3339)
	}
	return row
}

func payDateDisplay(e paymentHistoryEntry) string {
	if !e.hasPayDate {
		return ""
	}
	return e.payDate.Format("02/01/2006")
}

// filterHistoryEntries applies the Kỳ filter and the free-text search
// (employee name — diacritic-insensitive — CCCD, transfer note and bank
// transaction code).
func filterHistoryEntries(entries []paymentHistoryEntry, ky int, search string) []paymentHistoryEntry {
	out := entries
	if ky != 0 {
		filtered := make([]paymentHistoryEntry, 0, len(out))
		for _, e := range out {
			if e.ky == ky {
				filtered = append(filtered, e)
			}
		}
		out = filtered
	}
	if needle := foldVietnamese(search); needle != "" {
		filtered := make([]paymentHistoryEntry, 0, len(out))
		for _, e := range out {
			haystack := foldVietnamese(strings.Join([]string{e.employeeName, e.cccd, e.transferNote, e.bankCode}, "\x00"))
			if strings.Contains(haystack, needle) {
				filtered = append(filtered, e)
			}
		}
		out = filtered
	}
	return out
}

// foldVietnamese lower-cases and strips Vietnamese diacritics so "nguyen" and
// "Nguyễn" match the same row.
func foldVietnamese(s string) string {
	folded := norm.NFD.String(s)
	var b strings.Builder
	b.Grow(len(folded))
	for _, r := range folded {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r == 'đ' || r == 'Đ' {
			b.WriteRune('d')
			continue
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

// resolveHistoryMonth parses the YYYY-MM work month (empty = current month on
// the business clock).
func resolveHistoryMonth(month string) (time.Time, error) {
	if month == "" {
		now := clock.Now()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, clock.DefaultLocation), nil
	}
	parsed, err := time.ParseInLocation("2006-01", month, clock.DefaultLocation)
	if err != nil {
		return time.Time{}, domain.NewValidationError("Tháng không hợp lệ, định dạng đúng là YYYY-MM")
	}
	return parsed, nil
}
