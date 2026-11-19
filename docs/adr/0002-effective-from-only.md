# Effective-from only: no effective-to dates or window logic

Records carry at most a single effective-from date (`start_date` on assignments, `from_date` on payrates). There are no effective-to/end columns anywhere, and no window logic: an assignment row's existence IS the active assignment, enforced by `UNIQUE(project_id, employee_id)` — ending an assignment is a hard delete, changing it is an update of the single date.

## Why

- The donor's from/to pairs (`payment_schedule` windows, check-in activation windows, assignment `last_date`, project salary-period overrides) existed to serve the front-funding model's schedules and self-check-in rosters — all removed with that model.
- Window pairs invite boundary bugs (inclusive/exclusive ends, overlapping windows, "which window wins") that the fixed 4-Kỳ calendar makes unnecessary: the calendar is the only time structure.

## Considered options

- Keep from/to for future flexibility — rejected: YAGNI; every donor use of `to` was dead with the old model, and re-adding a column later is a trivial migration if a real need appears.

## Consequences

- "Is this assignment active?" is answered by row existence, not date comparison — rosters, active counts, and import resolvers must not reintroduce date-window predicates.
- Payrate history remains versioned by `from_date` only; the rate applying to a timesheet is the latest `from_date <= timesheet date`.
