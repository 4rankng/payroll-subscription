# Hard-delete-only: no soft delete anywhere

Records in the subscription payroll app are deleted with hard DELETEs; there is no `deleted_at` column, no `gorm.DeletedAt`, and no "active rows" query scoping. Uniqueness constraints are therefore absolute (e.g. `UNIQUE(cccd)` instead of unique-among-active-rows). Records referenced by salary history (timesheets, paid-Kỳ rows) cannot be deleted at all — FK RESTRICT returns an error, and "retiring" an employee means creating no new rows, never destroying history.

## Why

- Fresh schema, no legacy data: soft delete bought nothing and cost a second truth system (deleted-flag-aware indexes, generated helper columns, scope filters on every query).
- Billing reports must be reproducible from the rows that existed when they were generated; soft-delete scoping made "who was counted" mutable after the fact.
- `audit_logs` is the trail for what was deleted and by whom.

## Considered options

- Soft delete with active-row scoping (donor behaviour) — rejected: invariant complexity ("unique among active rows"), resurrection bugs via AutoMigrate-style tooling, and mutable counting history.
- Cascade delete of salary history — rejected: silently rewrites the basis of past billing reports.

## Consequences

- `db.Delete()` physically removes rows; deletes of referenced records must surface a clear business error instead of succeeding.
- The consolidated `schema.sql` contains no `deleted_at`; do not reintroduce it via model fields — AutoMigrate is disabled for this reason.
