---
title: "PAYSUB FR-6 Tong quan + local-dev hardening: ky-summary endpoint, schema/schema.sql fixes, integration suite rebuilt"
date: 2026-11-20
summary: "FR-6 dashboard shipped and verified in browser across admin/quanly/giamdoc; cold-run on fresh schema.sql exposed and fixed 4 real persistence/schema bugs; integration suite rebuilt to the live API (180 tests, 178 pass)"
---

# PAYSUB FR-6 Tong quan + local-dev hardening: ky-summary endpoint, schema/schema.sql fixes, integration suite rebuilt

## What happened
- FR-6 (PAYSUB-10): GET /billing/dashboard/ky-summary returns per-work-month distinct-employee counts per Kỳ + fee snapshot; KyFeeSummaryTable shipped on admin Tong quan (desktop+mobile) and partner portal; paid-series charts kept.
- Local-dev cold run (fresh schema.sql DB, browser-verified) exposed real bugs: schema.sql had inline decision annotations + missing index commas (cold deploy fails); EmployeePayCycle wrote a phantom cccd column (verification upserts broke); stale deleted_at/last_date/payment_schedule SQL 500'd summaries/project-users/grouped timesheets; gorm.DeletedAt on project_users/employee_users join models; timesheet-import active-scope leak made repeat uploads for the same project+month fail forever (fixed: terminal jobs release the scope, acceptance retries once).
- Dev logins admin/quanly/giamdoc/ketoan (Abc123) seeded in non-production; seed/admin.go dead code removed.
- Integration suite rebuilt to the live API: 25 removed-feature flows deleted, flow_billing added, flow_dashboard rewritten with removal pins; 180 tests / 178 pass / 0 fail.
- giamdoc role fixed end-to-end: casbin dashboard/metrics read rows, /partner route + PartnerLayout inner guard widened, login redirect corrected from deleted /accountant.

## Decision
- Seed only in dev (guard APP_ENV != production) per owner instruction.
- Weekly-BCC fixture flow parked: BCC LGD.xlsx is a May-2026 rateless file needing donor-era payrate semantics; upload-pipeline contract covered by flow_bcc_import.
- VN-only UI copy per founder: "Phí", "NV theo kỳ trả và phí".

## Next steps
- flow_bcc_weekly_import: re-enable when a PAYSUB-era fixture workbook exists.
- Billing GET routes are Authenticate-only (no casbin) — decide whether nhanvien should read them.
- giamdoc has no /api/v1/settings rows (settings UI is admin-only) — fine for read flows, revisit if directors need settings.

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
