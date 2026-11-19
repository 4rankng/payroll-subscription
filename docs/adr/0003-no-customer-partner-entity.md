# No customer/partner entity: single standalone deployment

The system has no customer or partner domain object — no `customers` table, no `partner_company` setting, no tenant scoping. One deployment serves one paying company; billing reports are plain monthly exports (work month + fee + per-Kỳ lines) whose sending and payment collection happen entirely outside the system. `Project.client_name` survives only as a per-project site label.

## Why

- The product bills one paying company per deployment; a tenant entity modeled a relationship that has exactly one instance and no lifecycle in the system.
- Removing the concept deleted FR-5, the `partner_company` setting, and all "customer vs deployment" counting caveats — distinct-employee counting is simply per-database.

## Considered options

- Multi-tenant SaaS with a `customers` table and per-tenant pricing — rejected at grill (Q1): subscription economics today are single-customer; if multi-tenancy ever becomes real, it is a deliberate one-way migration, not a guess baked in early.
- Keep `partner_company` as a report header label — rejected: unused in the new model; the report title needs only the work month and fee.

## Consequences

- Role names are internal (`admin/quanly/giamdoc/nhanvien`); nothing in the schema, casbin policy, or UI may reintroduce a partner/customer actor.
- Onboarding a new paying company = deploy a new instance (schema + settings seed), not a row insert.
