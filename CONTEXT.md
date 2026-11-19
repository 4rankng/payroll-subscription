# Payroll Subscription (PAYSUB) — Domain Glossary

Subscription payroll system: the paying company funds salaries externally; the platform tracks pay cycles, counts distinct employees, and bills a flat fee.

## Language

**Kỳ** (plural: Kỳ):
One fixed pay cycle of the month. There are exactly 4 per month with fixed windows (1–7, 8–14, 15–21, 22–end) and fixed pay dates (10th, 17th, 24th, 1st of next month). Not configurable.
_Avoid_: round, period, cycle, chu kỳ

**Paying company**:
The company that funds salaries from its own bank account and uploads the transfer result. There is exactly one per deployment; it is not modeled as an entity.
_Avoid_: customer, partner, client (as an entity — `client_name` on a project is only a site label)

**Distinct employee (of a Kỳ)**:
An employee counted once per Kỳ, identified by CCCD. The same employee in N Kỳ bills N times (once each). The billing atom: one employee × one Kỳ.
_Avoid_: headcount, unique employee

**Timesheet**:
One employee-day record with its salary amount and approval state. The only source of salary amounts; the substrate for counting.
_Avoid_: attendance (removed concept), work log

**Customer-paid**:
The state of a timesheet whose salary was verified as paid via an uploaded transfer result (`payment_status='paid'`). The only counting gate. Written exclusively by verification.
_Avoid_: paid (unqualified), externally-paid

**Transfer result (upload)**:
The bank transfer evidence file the paying company uploads for one Kỳ. Processed row-by-row; identical content (same hash) is never reprocessed.
_Avoid_: statement, sao kê, receipt

**Verification**:
Row-by-row matching of an upload against approved timesheets of that Kỳ. Matched rows become customer-paid; failed rows are reported per-row and never block other rows.
_Avoid_: reconciliation, đối soát, settlement

**Assignment**:
One project–employee row. Its existence means the employee is active on that project; there are no effective-to dates and no window logic.
_Avoid_: contract, allocation, active roster entry

**Effective from**:
The single start date a record may carry. No record has an effective-to date.
_Avoid_: valid until, end date, last_date

**Billing report**:
The month-end Excel artifact: distinct employees per Kỳ × fee, with the fee snapshot embedded. The platform's terminal deliverable; sending it and collecting payment happen outside the system.
_Avoid_: invoice, hóa đơn, payment report

**Roles**:
`admin` (full control), `quanly` (paying-company manager), `giamdoc` (director — billing-report export), `nhanvien` (read-only own pay history).
_Avoid_: partner, customer, accountant, ketoan, adv_partner

**Hard delete**:
Every delete physically removes the row. Records referenced by salary history cannot be deleted at all.
_Avoid_: soft delete, archive, trash
