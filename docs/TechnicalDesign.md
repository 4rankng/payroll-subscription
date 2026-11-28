# Technical Design — PAYSUB Refactor (Database + Codebase)

| Field | Value |
|---|---|
| Status | Draft v12 (single consolidated schema; hard DELETE only; effective-from only; RESTRICT on referenced records — D18; roles admin/quanly/ketoan/giamdoc/nhanvien — D19 as amended by D28; Zalo ZNS removed — D20; no customer/partner concept — D21; dead logic purged — D22; FR-6 Tổng quan + FR-7 sao kê lương tuần export/email; founder scope note 2026-11-27: lương tuần only, no OnePay/9Pay — D27. Refreshed against donor HEAD `090e4463`, window 2026-11-17…26) |
| Date | 2026-11-27 (donor + founder-note refresh; originally drafted 2026-11-19) |
| Input | `NewAppIdea.md` (PRD v13, decisions D1–D28, no open questions) |
| Scope | How to refactor this monorepo in place into the subscription model: schema surgery + demolition sequencing + new-model build. Requirements are in the PRD; this doc is the execution design. |
| Board | plan.tingting.vip/tingtingsoft/PAYSUB (tickets PAYSUB-1…9) |
| Session rule | Requirements/design first; PAYSUB execution (Track A billing, PAYSUB-10/11/1344) started 2026-11-27 under accepted founder decisions |

## 1. Strategy

In-place refactor of the donor monorepo. No production data migration exists (greenfield product), so the design optimizes for: every commit compiles, `make api-test` stays green between waves, and destructive schema steps land last.

Deployment is from scratch, so the database is delivered as one consolidated script (§3) — there are no incremental migrations. Three tracks, interleaved:

| Track | What | Order rationale |
|---|---|---|
| A — Build | New-model slices (fee setting, upload pipeline, verification, Kỳ read model, billing report) on kept infrastructure | Goes first: nothing new depends on removed code |
| B — Demolish | Old-model removal in 5 leaf-first waves (UI → transport → services/jobs → domain/repos → schema) | Leaf-first so each wave compiles without the next |
| C — Schema | ONE consolidated `schema.sql` creating every table, applied once on first deploy | Fresh deployment: no ALTER/DROP chains, no backup gate, no migration history |

Track A can start immediately. Waves 1–2 (UI/transport) can run in parallel with Track A only if container.go is left untouched until Wave 2 starts — container wiring is the single serialization point.

## 2. Schema Delivery — Single Consolidated Script

Decision: the database is created by **one SQL script** — `backend/migrations/schema.sql` (content in §3) — applied once by the deploy pipeline on first boot.

- Donor migrations `backend/migrations/001…146` are **not carried over**; the whole numbered-migration history is replaced by this single baseline. `schema_migrations` table is not recreated.
- gorm `AutoMigrate` is **removed** from `bootstrap/infrastructure/init.go`: `schema.sql` is the only schema authority (running both would let AutoMigrate resurrect dropped tables/columns and drift from the script).
- Seed row `subscription_fee_per_employee_per_ky_vnd` ships inside the script (`partner_company` removed, D21); runtime seed code in `backend/internal/seed/` stays only for non-schema defaults.
- Donor DDL source: the running dev database (`payroll-mysql`, `SHOW CREATE TABLE`) — the script below is that DDL with the removals applied, so column types/indexes match what the code actually runs on today. Re-verified 2026-11-27: no donor migrations landed 2026-11-17…26, so the DDL base is unchanged at donor HEAD `090e4463`.

## 3. Final Schema (consolidated, 23 tables)

Surgery applied to donor DDL:

| Table | Removed vs donor | Decision ref |
|---|---|---|
| `employees` | `bank_id`, `bank_account_*` (4 cols), `bank_account_validated_at`; FK `fk_employees_bank` | OnePay verification dies |
| `projects` | `salary_period_from/to` (+ their CHECK constraints), `is_flexible`, `geofence_gates`, `geofence_radius_meters`, `shift_names`, `pending_receivable_vnd`, `total_received_vnd` | D8, D12, D14; `total_payout_vnd`/`pending_payable_vnd` kept and re-meant to customer-paid totals (D11) |
| `project_employees` | `payment_schedule*` (3), `check_in_*` (4), `advance_request_enabled`, **`last_date`** (effective-to — D17) + all window/`payment_schedule` indexes; **`UNIQUE(project_id, employee_id)`** added — an assignment row IS the active assignment (no window logic) | D8, D12, D17; `start_date` stays as the only effective-from |
| `timesheets` | `revenue_receivable`, `revenue_paid`, `transaction_id` + its index | D5/D14; `payment_status/payment_date/paid_amount/paid_at/payment_reference` kept, re-meant customer-paid (D11) |
| `timesheets` | `force_payroll` (bulk-transfer forcing) + `idx_timesheets_force_payroll`; `'failed'` enum value — `payment_status` shrunk to `pending/paid/cancelled` | D22 (fresh schema keeps no dead logic). `payment_reference` **kept, re-meant** (grill correction): stores the bank transaction reference from the customer's uploaded transfer result; written only by verification alongside `paid` |
| EDIT-REQUEST REMOVAL | Drop table `timesheet_edit_requests` + FK/indexes; `timesheets` drops `allowed_edit`, `request_edit_id` (+ `idx_timesheets_request_edit_id`, `idx_timesheets_access_control_edit`, `idx_timesheets_updated_at_filter` which reference them) — D25; schema.sql already excludes them (keep it that way) |
| `payrates` | `live_from_date` dropped → `UNIQUE(project_id, from_date)` (soft-delete row below). Timeline semantics re-anchored at donor HEAD (2026-11-18…23): a config with ANY linked timesheet is frozen (update/delete blocked, create-new only); creates may backdate only to a strictly earlier unoccupied date (no mutable timesheet on/after); `payrate_json` shift buckets may overlap — rate lookup is exact-key per shift name, overlap validation removed (`domain/payrate.go`) | D17/D18 — donor behavior already matches (referenced rows immutable, effective-from only) |
| ALL kept tables | `deleted_at` columns, every `deleted_at`-containing index, gorm soft-delete machinery (D16): deletes are hard DELETEs. Generated soft-delete helpers dropped: `settings.active_key` → `UNIQUE(key)`. Rebuilt uniques: `users` UNIQUE(cccd)/UNIQUE(mobile); `employees` UNIQUE(email)/UNIQUE(cccd); `project_users` UNIQUE(project_id, user_id); `employee_users` UNIQUE(employee_id, user_id) | D16 |
| excluded entirely | `attendances`, `attendance_failed_attempts`, `banks`, `zns_messages`, ledger/wallet/loan/advance/bulk-transfer/gateway/ads/API-key/health tables | D13/D14/D15/D20 + PRD §6.3 |
| added | `transfer_uploads`, `employee_pay_cycles`, `billing_reports`, `billing_report_lines` | FR-1…FR-4 |

```sql
-- PAYSUB consolidated schema — single source of truth, applied once on first deploy.
-- Fresh deployment: donor migrations 001-146 are NOT carried over.
-- NO soft delete anywhere: no deleted_at columns, no gorm.DeletedAt — deletes are hard DELETEs (PRD D16).
-- EFFECTIVE-FROM ONLY: no effective-to columns/logic anywhere (PRD D17); an assignment row IS the active assignment.
-- Attendance / ledger / wallet / loans / advances / bulk-transfer / gateway tables are
-- deliberately absent (PRD v11 §6.3, D1-D15). gorm AutoMigrate must stay OFF:
-- this script is the only schema authority.
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS=0;

CREATE DATABASE IF NOT EXISTS `payroll_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
USE `payroll_db`;

CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `password` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `fullname` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `role` enum('admin','quanly','ketoan','giamdoc','nhanvien') NOT NULL DEFAULT 'nhanvien' (D19 as amended by D28),
  `last_login` datetime(3) DEFAULT NULL,
  `tokens_invalid_before` datetime(3) DEFAULT NULL COMMENT 'when set, any JWT issued before this instant is rejected on validate; used by RevokeUserTokens',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `cccd` varchar(20) DEFAULT NULL COMMENT '''Citizen ID for admin/quanly login''',
  `mobile` varchar(15) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  KEY `idx_users_email_nonunique` (`email`),
  KEY `idx_users_email` (`email`),
  KEY `idx_users_last_login` (`last_login`)
UNIQUE KEY `uq_users_cccd` (`cccd`),
UNIQUE KEY `uq_users_mobile` (`mobile`),
KEY `idx_users_role` (`role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `employees` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `fullname` varchar(255) NOT NULL,
  `email` varchar(255) DEFAULT NULL,
  `cccd` varchar(255) NOT NULL COMMENT '''Citizen ID - Can cong cong dan (12 digits)''',
  `address` text,
  `mobile` varchar(15) DEFAULT NULL,
  `date_of_birth` date DEFAULT NULL,
  `created_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `search_normalized` varchar(1000) GENERATED ALWAYS AS (lower(concat(coalesce(`fullname`,_utf8mb4''),_utf8mb4' ',coalesce(`email`,_utf8mb4''),_utf8mb4' ',coalesce(`cccd`,_utf8mb4''),_utf8mb4' ',coalesce(`mobile`,_utf8mb4'')))) VIRTUAL,
  `user_id` bigint unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_employees_creator` (`created_by`),
  KEY `idx_employees_user_id` (`user_id`),
  KEY `idx_employees_created_at` (`created_at`),
  KEY `idx_employees_search` (`fullname`,`cccd`),
  KEY `idx_employees_mobile` (`mobile`),
  KEY `idx_employees_search_normalized` (`search_normalized`(255)),
  CONSTRAINT `fk_employees_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_employees_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_employees_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
UNIQUE KEY `uq_employees_email` (`email`),
UNIQUE KEY `uq_employees_cccd` (`cccd`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `projects` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `client_name` varchar(255) DEFAULT NULL COMMENT '''e.g. Nha may san xuat hoa my pham VERICO''',
  `name` varchar(255) NOT NULL COMMENT '''e.g. San xuat xa phong''',
  `code` varchar(255) DEFAULT NULL COMMENT '''Initials of client_name and YYMM of created_at date or manually input by users''',
  `description` text COMMENT '''Rich text format supported''',
  `total_payout_vnd` decimal(15,2) NOT NULL DEFAULT '0.00' COMMENT '''Total paid to employees to date''',
  `pending_payable_vnd` decimal(15,2) NOT NULL DEFAULT '0.00' COMMENT '''Pending payment to employees to date''',
  `project_status` enum('draft','active','paused','completed','cancelled') NOT NULL DEFAULT 'draft',
  `created_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `off_days` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '''Bitmask of weekly off days: bit0=Sun,bit1=Mon,...,bit6=Sat. Default 0 = no fixed off days''',
  `owner_revoked_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_projects_code` (`code`),
  KEY `fk_projects_creator` (`created_by`),
  KEY `idx_projects_created_at` (`created_at`),
  KEY `idx_projects_search_name` (`name`),
  KEY `idx_projects_search_code` (`code`),
  KEY `idx_projects_search_client` (`client_name`),
  KEY `idx_projects_status_created_by` (`project_status`,`created_by`),
  KEY `idx_projects_created_by_dates` (`created_by`,`created_at`,`updated_at`),
  CONSTRAINT `fk_projects_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `project_employees` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint unsigned NOT NULL,
  `employee_id` bigint unsigned NOT NULL,
  `employee_name` varchar(255) NOT NULL,
  `employee_cccd` varchar(255) NOT NULL COMMENT '''Citizen ID - Can cong cong dan (12 digits)''',
  `employee_code` varchar(255) DEFAULT NULL COMMENT '''Factory-assigned code, optional''',
  `position` varchar(100) NOT NULL DEFAULT 'phổ thông' COMMENT '''Employee position for payrate calculation''',
  `start_date` date NOT NULL,
  `created_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_project_employees_creator` (`created_by`),
  KEY `fk_project_employees_project` (`project_id`),
  KEY `fk_project_employees_employee` (`employee_id`),
  CONSTRAINT `fk_project_employees_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_project_employees_employee` FOREIGN KEY (`employee_id`) REFERENCES `employees` (`id`),
  CONSTRAINT `fk_project_employees_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`)
UNIQUE KEY `uq_project_employees_pair` (`project_id`,`employee_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `project_users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `granted_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_project_users_user` (`user_id`),
  KEY `idx_project_users_granted_by` (`granted_by`),
  CONSTRAINT `fk_project_users_granted_by` FOREIGN KEY (`granted_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_project_users_granted_by_user` FOREIGN KEY (`granted_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_project_users_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`),
  CONSTRAINT `fk_project_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
UNIQUE KEY `uq_project_users_project_user` (`project_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `employee_users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `employee_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `granted_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_employee_users_employee_id` (`employee_id`),
  KEY `idx_employee_users_user_id` (`user_id`),
  KEY `fk_employee_users_granted_by_user` (`granted_by`),
  CONSTRAINT `fk_employee_users_employee` FOREIGN KEY (`employee_id`) REFERENCES `employees` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_employee_users_granted_by` FOREIGN KEY (`granted_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_employee_users_granted_by_user` FOREIGN KEY (`granted_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_employee_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
UNIQUE KEY `uq_employee_users_pair` (`employee_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `payrates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint unsigned NOT NULL,
  `payrate_json` json NOT NULL COMMENT '''custom format per project in format paytype: vnd_per_hour e.g. {"normal": 10000, "overtime": 15000, "weekend": 20000, "holiday": 25000}''',
  `from_date` date NOT NULL,
  `created_by` bigint unsigned NOT NULL COMMENT '''User who created the payrate''',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_payrates_project` (`project_id`),
  KEY `fk_payrates_created_user` (`created_by`),
  CONSTRAINT `fk_payrates_created_user` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_payrates_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`)
UNIQUE KEY `uq_payrates_project_from_date` (`project_id`,`from_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `assets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `filename` varchar(255) NOT NULL,
  `file_path` varchar(500) NOT NULL,
  `upload_type` varchar(50) NOT NULL DEFAULT 'general',
  `checksum` varchar(64) DEFAULT NULL,
  `uploaded_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `metadata` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_assets_checksum` (`checksum`),
  KEY `fk_assets_uploader` (`uploaded_by`),
  CONSTRAINT `fk_assets_uploader` FOREIGN KEY (`uploaded_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `timesheets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint unsigned NOT NULL,
  `employee_id` bigint unsigned NOT NULL,
  `payrate_id` bigint unsigned NOT NULL COMMENT '''Reference to payrate config used''',
  `date` date NOT NULL,
  `hours_worked` decimal(4,2) NOT NULL DEFAULT '0.00' COMMENT '''Hours worked (0.00-99.99)''',
  `paytype` varchar(100) NOT NULL COMMENT '''Flexible paytype from payrate config''',
  `payrate` bigint NOT NULL COMMENT '''VND amount from payrates config''',
  `amount` bigint NOT NULL DEFAULT '0',
  `timesheet_status` enum('pending_approval','approved','rejected') NOT NULL DEFAULT 'pending_approval',
  `payment_status` enum('pending','paid','cancelled') NOT NULL DEFAULT 'pending' COMMENT 'customer-paid gate: paid written only by upload verification (D11)',
  `payment_date` date DEFAULT NULL,
  `paid_amount` bigint DEFAULT '0' COMMENT '''Total amount paid for this timesheet''',
  `paid_at` datetime(3) DEFAULT NULL,
  `payment_reference` varchar(255) DEFAULT NULL COMMENT '''Bank transaction reference from the customer''s uploaded transfer result; written by verification only (D11/D22)''',
  `created_by` bigint unsigned NOT NULL,
  `approved_by` bigint unsigned DEFAULT NULL COMMENT '''Admin who approved the timesheet''',
  `approved_at` datetime(3) DEFAULT NULL,
  `rejection_reason` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_timesheets_payrate` (`payrate_id`),
  KEY `idx_timesheets_project_employee_date` (`project_id`,`employee_id`,`date`),
  KEY `idx_timesheets_created_by` (`created_by`),
  KEY `idx_timesheets_approved_by` (`approved_by`),
  KEY `idx_timesheets_project_created` (`project_id`,`created_at`),
  KEY `idx_timesheets_payment_paid_at` (`payment_status`,`paid_at`),
  KEY `idx_timesheets_employee_payment_paid` (`employee_id`,`payment_status`,`paid_at`),
  CONSTRAINT `fk_timesheets_approved_user` FOREIGN KEY (`approved_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_timesheets_created_user` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_timesheets_employee` FOREIGN KEY (`employee_id`) REFERENCES `employees` (`id`),
  CONSTRAINT `fk_timesheets_payrate` FOREIGN KEY (`payrate_id`) REFERENCES `payrates` (`id`),
  CONSTRAINT `fk_timesheets_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`),KEY `idx_timesheets_employee_date` (`employee_id`,`date`),
KEY `idx_timesheets_date_status` (`date`,`timesheet_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `timesheet_edit_requests` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `timesheet_id` bigint unsigned NOT NULL COMMENT '''Foreign key to timesheets table''',
  `status` enum('pending','approved','rejected') CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'pending' COMMENT '''Request status''',
  `requested_by` bigint unsigned NOT NULL COMMENT '''User ID who requested the edit''',
  `approved_by` bigint unsigned DEFAULT NULL COMMENT '''Admin user ID who approved the request''',
  `rejected_by` bigint unsigned DEFAULT NULL COMMENT '''Admin user ID who rejected the request''',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ter_timesheet_id` (`timesheet_id`),
  KEY `idx_ter_status` (`status`),
  KEY `idx_ter_requested_by` (`requested_by`),
  KEY `idx_ter_created_at` (`created_at` DESC),
  KEY `fk_timesheet_edit_requests_rejected_user` (`rejected_by`),
  KEY `fk_timesheet_edit_requests_approved_user` (`approved_by`),
  CONSTRAINT `fk_ter_approved_by` FOREIGN KEY (`approved_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_ter_rejected_by` FOREIGN KEY (`rejected_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_ter_requested_by` FOREIGN KEY (`requested_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_ter_timesheet` FOREIGN KEY (`timesheet_id`) REFERENCES `timesheets` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_timesheet_edit_requests_approved_user` FOREIGN KEY (`approved_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_timesheet_edit_requests_rejected_user` FOREIGN KEY (`rejected_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_timesheet_edit_requests_requested_user` FOREIGN KEY (`requested_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_timesheet_edit_requests_timesheet` FOREIGN KEY (`timesheet_id`) REFERENCES `timesheets` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Tracks requests to edit approved timesheets';

CREATE TABLE `timesheet_import_jobs` (
  `asset_id` bigint unsigned NOT NULL,
  `project_id` bigint unsigned NOT NULL,
  `for_month` char(7) NOT NULL,
  `uploaded_by` bigint unsigned NOT NULL,
  `uploader_role` varchar(32) NOT NULL,
  `include_flexible_employees` tinyint(1) NOT NULL DEFAULT '0',
  `add_on_top` tinyint(1) NOT NULL DEFAULT '0',
  `status` enum('pending','processing','completed','failed') NOT NULL DEFAULT 'pending',
  `idempotency_key` varchar(128) NOT NULL,
  `request_fingerprint` char(64) NOT NULL,
  `active_scope_key` varchar(64) DEFAULT NULL,
  `attempt` int unsigned NOT NULL DEFAULT '0',
  `lease_expires_at` datetime(3) DEFAULT NULL,
  `started_at` datetime(3) DEFAULT NULL,
  `processed_at` datetime(3) DEFAULT NULL,
  `audit_logged_at` datetime(3) DEFAULT NULL,
  `last_error` text,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`asset_id`),
  UNIQUE KEY `uq_timesheet_import_idempotency` (`uploaded_by`,`idempotency_key`),
  UNIQUE KEY `uq_timesheet_import_active_scope` (`active_scope_key`),
  KEY `idx_timesheet_import_recovery` (`status`,`lease_expires_at`),
  KEY `fk_timesheet_import_project` (`project_id`),
  CONSTRAINT `fk_timesheet_import_asset` FOREIGN KEY (`asset_id`) REFERENCES `assets` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_timesheet_import_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`),
  CONSTRAINT `fk_timesheet_import_uploader` FOREIGN KEY (`uploaded_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;


CREATE TABLE `settings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(100) NOT NULL,
  `value` text,
  `value_type` enum('string','number','boolean','json') DEFAULT 'string',
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
UNIQUE KEY `uq_settings_key` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `notifications` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sender_id` bigint unsigned NOT NULL COMMENT '''User who initiated/sent the notification''',
  `recipient_id` bigint unsigned DEFAULT NULL COMMENT '''User who receives the notification (null for emails to external recipients)''',
  `type` varchar(50) NOT NULL,
  `channel` varchar(20) NOT NULL DEFAULT 'push',
  `title` varchar(255) NOT NULL,
  `message` text NOT NULL,
  `content_type` varchar(20) NOT NULL DEFAULT 'plain_text',
  `email_recipients` text COMMENT '''JSON array of email recipients for email notifications''',
  `resend_message_id` varchar(255) DEFAULT NULL COMMENT '''Resend email message ID for tracking''',
  `read_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `metadata` text COMMENT '''JSON object with metadata for payroll emails''',
  PRIMARY KEY (`id`),
  KEY `idx_notifications_channel` (`channel`),
  KEY `idx_sender_id` (`sender_id`),
  KEY `idx_recipient_id` (`recipient_id`),
  KEY `idx_notifications_content_type` (`content_type`),
  KEY `idx_notifications_sender_id` (`sender_id`),
  KEY `idx_notifications_recipient_id` (`recipient_id`),
  CONSTRAINT `fk_notifications_recipient` FOREIGN KEY (`recipient_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_notifications_sender` FOREIGN KEY (`sender_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `push_subscriptions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `endpoint` varchar(500) NOT NULL,
  `p256dh` varchar(200) NOT NULL,
  `auth` varchar(100) NOT NULL,
  `device_type` varchar(20) NOT NULL DEFAULT 'web',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_push_sub_user_endpoint` (`user_id`,`endpoint`(191)),
  KEY `idx_push_sub_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `message` text CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `action` varchar(50) NOT NULL,
  `entity_type` varchar(50) NOT NULL,
  `entity_id` bigint unsigned DEFAULT NULL,
  `ip_address` varchar(45) DEFAULT NULL,
  `metadata` json DEFAULT NULL,
  `browser` varchar(100) DEFAULT NULL,
  `platform` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_audit_logs_user_id` (`user_id`),
  KEY `idx_audit_logs_created_at` (`created_at`),
  KEY `idx_action_entity` (`action`,`entity_type`),
  KEY `idx_audit_logs_entity_id` (`entity_id`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `blacklisted_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `token_jti` varchar(255) NOT NULL COMMENT '''JWT ID''',
  `user_id` bigint unsigned NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `blacklisted_at` datetime(3) DEFAULT NULL,
  `reason` enum('logout','security','expired','admin_revoke') DEFAULT 'logout',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_blacklisted_tokens_token_jti` (`token_jti`),
  KEY `fk_blacklisted_tokens_user` (`user_id`),
  KEY `idx_blacklisted_tokens_jti_expires` (`token_jti`,`expires_at`),
  KEY `idx_blacklisted_tokens_expires` (`expires_at`),
  KEY `idx_blacklisted_tokens_reason_date` (`reason`,`blacklisted_at`),
  KEY `idx_blacklisted_tokens_user_expires` (`user_id`,`expires_at`),
  CONSTRAINT `fk_blacklisted_tokens_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `outbox_events` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `event_id` varchar(36) NOT NULL COMMENT 'UUID of the event for idempotency',
  `event_type` varchar(100) NOT NULL COMMENT 'Type of event (e.g., UploadVerified, BillingReportGenerated)',
  `aggregate_type` varchar(50) NOT NULL COMMENT 'Type of aggregate (transfer_upload, employee_pay_cycle, billing_report)',
  `aggregate_id` bigint unsigned NOT NULL COMMENT 'ID of the aggregate',
  `payload` json NOT NULL COMMENT 'Event payload as JSON',
  `status` varchar(20) NOT NULL DEFAULT 'pending' COMMENT 'pending, published, failed',
  `retry_count` int NOT NULL DEFAULT '0' COMMENT 'Number of retry attempts',
  `max_retries` int NOT NULL DEFAULT '5' COMMENT 'Maximum number of retries',
  `last_error` text COMMENT 'Last error message if failed',
  `published_at` timestamp NULL DEFAULT NULL COMMENT 'When the event was successfully published',
  `next_retry_at` timestamp NULL DEFAULT NULL COMMENT 'When to retry next (exponential backoff)',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_event_id` (`event_id`),
  KEY `idx_status` (`status`),
  KEY `idx_aggregate` (`aggregate_type`,`aggregate_id`),
  KEY `idx_next_retry` (`status`,`next_retry_at`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Transactional outbox for reliable event publishing';

CREATE TABLE `outbox_event_handlers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `outbox_event_id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `handler_type` varchar(100) NOT NULL,
  `status` enum('pending','processing','completed','failed') NOT NULL DEFAULT 'pending',
  `retry_count` int NOT NULL DEFAULT '0',
  `max_retries` int NOT NULL DEFAULT '5',
  `last_error` text,
  `started_at` datetime(3) DEFAULT NULL,
  `completed_at` datetime(3) DEFAULT NULL,
  `next_retry_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_outbox_event_id` (`outbox_event_id`),
  KEY `idx_handler_type` (`handler_type`),
  KEY `idx_status` (`status`),
  KEY `idx_next_retry_at` (`next_retry_at`),
  KEY `idx_outbox_event_handler` (`outbox_event_id`,`handler_type`),
  CONSTRAINT `fk_outbox_event_handlers_outbox_event_id` FOREIGN KEY (`outbox_event_id`) REFERENCES `outbox_events` (`event_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `transfer_uploads` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cycle` tinyint unsigned NOT NULL COMMENT 'Pay cycle 1..4 (clock.TimesheetKyCount)',
  `work_month` char(7) NOT NULL COMMENT 'YYYY-MM the Ky belongs to',
  `file_hash` char(64) NOT NULL COMMENT 'SHA-256 of the upload; idempotency key (settlement_uploads pattern)',
  `original_filename` varchar(255) NOT NULL,
  `asset_id` bigint unsigned DEFAULT NULL COMMENT 'Stored file (assets)',
  `uploaded_by` bigint unsigned NOT NULL,
  `status` enum('pending','verified','rejected') NOT NULL DEFAULT 'pending',
  `reject_reason` varchar(500) DEFAULT NULL,
  `verified_at` datetime(3) DEFAULT NULL,
  `verified_by` bigint unsigned DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_transfer_uploads_file_hash` (`file_hash`),
  KEY `idx_transfer_uploads_cycle_month` (`cycle`,`work_month`,`status`),
  CONSTRAINT `fk_transfer_uploads_asset` FOREIGN KEY (`asset_id`) REFERENCES `assets` (`id`),
  CONSTRAINT `fk_transfer_uploads_uploader` FOREIGN KEY (`uploaded_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `employee_pay_cycles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cycle` tinyint unsigned NOT NULL,
  `work_month` char(7) NOT NULL,
  `employee_id` bigint unsigned NOT NULL,
  `project_id` bigint unsigned NOT NULL,
  `timesheet_id` bigint unsigned NOT NULL,
  `source_upload_id` bigint unsigned NOT NULL,
  `paid_amount` bigint NOT NULL DEFAULT '0',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_employee_pay_cycle` (`employee_id`,`cycle`,`work_month`),
  KEY `idx_employee_pay_cycles_month_cycle` (`work_month`,`cycle`),
  CONSTRAINT `fk_employee_pay_cycles_employee` FOREIGN KEY (`employee_id`) REFERENCES `employees` (`id`),
  CONSTRAINT `fk_employee_pay_cycles_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`),
  CONSTRAINT `fk_employee_pay_cycles_timesheet` FOREIGN KEY (`timesheet_id`) REFERENCES `timesheets` (`id`),
  CONSTRAINT `fk_employee_pay_cycles_upload` FOREIGN KEY (`source_upload_id`) REFERENCES `transfer_uploads` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `billing_reports` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `work_month` char(7) NOT NULL,
  `fee_per_employee_per_ky_vnd` int NOT NULL COMMENT 'Fee snapshot at generation time (FR-4)',
  `total_distinct` int NOT NULL DEFAULT '0',
  `total_vnd` bigint NOT NULL DEFAULT '0',
  `excel_path` varchar(500) DEFAULT NULL,
  `generated_by` bigint unsigned NOT NULL,
  `generated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_billing_reports_work_month` (`work_month`),
  CONSTRAINT `fk_billing_reports_generator` FOREIGN KEY (`generated_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `billing_report_lines` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `report_id` bigint unsigned NOT NULL,
  `cycle` tinyint unsigned NOT NULL,
  `distinct_count` int NOT NULL DEFAULT '0',
  `amount_vnd` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_billing_report_line` (`report_id`,`cycle`),
  CONSTRAINT `fk_billing_report_lines_report` FOREIGN KEY (`report_id`) REFERENCES `billing_reports` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Seed settings (subscription fee)
INSERT INTO `settings` (`key`,`value`,`value_type`) VALUES
  ('subscription_fee_per_employee_per_ky_vnd','10000','number');


SET FOREIGN_KEY_CHECKS=1;
```

## 4. Track A — Build the New Model (PAYSUB-2…6)

| Step | Design | Reuses |
|---|---|---|
| A1 Fee setting | Add key to `services/config/settings_config.go` (constant + getter + validation: integer ≥ 0), seed row | Existing settings cache pattern |
| A2 Upload pipeline | Format: the existing weekly-payment **result Excel** — rows keyed by CCCD (donor matching: `bcc_import_weekly_payment.go:94-98` — the row's own CCCD, then the workbook's STK CCCD for that name, then the project-employee name). Parser accepts POURIN weekly-payment templates since 2026-11-18 (row-8 identity labels, `Lương/8h` spellings, TC/LE shift codes, empty day-31 scaffolding skipped) — FR-2 inherits that tolerance. Job-based ingest: `POST /api/v1/billing/uploads` (multipart) → job row → parse → verify. Job states, locks, result/audit copied from the bcc_import file family; FileHash idempotency from `domain/settlement_upload.go` (identical hash → return prior result) | bcc_import lifecycle files |
| A3 Verification | **Per-row processing (current donor behaviour)**: each parsed row (CCCD + amount) is processed independently against approved timesheets whose `KyFromWorkDay(Date)` = upload's Kỳ and month. Matched row → same-tx `timesheets SET payment_status='paid', payment_date=…` + upsert `employee_pay_cycles` (amount summed per employee-Kỳ). Failed row (unknown CCCD, no timesheet) → per-row error, upload continues; upload status `verified` when ≥1 row processed, `rejected` only when every row failed → then all-admin push (D10) via the `WalletBalanceAlertService` fan-out mechanism (keep the push util, delete the alert job). Timesheets not in the file stay `pending` | clock/pay_cycle, bcc validation style, settlement per-row iteration |
| A4 Kỳ read model | **One row per employee per Kỳ** (founder: the billing atom): `UNIQUE(employee_id, cycle, work_month)`; `paid_amount` = SUM of the employee's customer-paid timesheet amounts across all projects in that Kỳ. Monthly billing = rows of the month × fee; distinct-CCCD is the row count | — |
| A5 Billing report | Scheduler job `generate_monthly_billing_report` (cron `0 9 1 * *`, replaces the removed `daily_receivable_reconciliation` slot): builds `billing_reports` + lines with fee snapshot → renders Excel via `app/services/excel/` with the D24 layout — **Summary sheet** (per-Kỳ distinct × fee + total) + **one sheet per project** listing that project's payment history (employee, CCCD, Kỳ, amount, payment date/reference) → stores file → push/portal notice. Manual regenerate endpoint for admin | excel exporters, scheduler pattern |
| A6 UI | Quanly portal: Upload page (Kỳ selector + file + status/history) and Report page (month list → Excel download) — replaces `PaymentHistoryPage` content. Admin: fee field in reduced Settings `general` tab | quanly portal pages (`pages/partner/` legacy dir) |
| A7 Tổng quan dashboard (FR-6) | Endpoints: GET /billing/dashboard/paid-series?project_id=&from=&to= (per-day paid-employee count per project, from customer-paid timesheets) and paid-amount-series (same substrate, SUM(amount)); per-month view reuses GET /billing/ky-paid-records summary + fee setting (BR-1). UI replaces the removed FinancialChart widgets on the Tổng quan page (desktop + mobile) | timesheet repo + clock.KyFromWorkDay + existing chart components |

Track A gates (repo rules): new integration tests in `backend/tests/integration` (upload → verify → count → report happy path + mismatch rejection + idempotent re-upload); UI parity desktop 1280 / mobile 390 for admin + quanly; `make api-test`, `pnpm lint && pnpm type-check`.

## 5. Track B — Demolition Waves (leaf-first)

Each wave ends with the gate: `go build ./... && go test ./... -race` + `pnpm lint && pnpm type-check` + `make api-test`. One wave per merge to main.

### Wave 1 — Frontend UI (PAYSUB-7/8/9 UI halves)

- Delete routes + lazy imports in `frontend/src/App.tsx`: `/admin/ledger` + `/transactions` redirect (280–281), `/admin/cron-health`, `/admin/system-health` (291–293), accountant, loans, ads, advance-payments, manual-disbursement pages; remove `settings?tab=fee-config|ads|api|zalo` panels from `pages/admin/SettingsPage/index.tsx` + `SettingsTabList.tsx` (ZNS removed, D20); drop `pages/ZaloResetPassword.tsx` (password reset only); remove `/admin/zns-tracking` route + `pages/admin/ZnsTrackingPage` + `pages/mobile/admin/ZnsTrackingPage` + sidebar entry (`AdminSidebar.tsx:91`) + `hooks/api/useZnsMessages.ts` (ZNS removed, D20).
- Delete dirs: `components/ledger/`, `components/transaction/`, `components/advance-payment/`, `components/cron-health/`, `components/admin/WeeklyPaymentFeeSchedule/`, `components/settings/{ApiKeysSection,McpApiKeysSection,OpsApiKeysSection,AdBannerSection,FeeScheduleSection,DisbursementFeeScheduleSection}`, `components/admin-dashboard/FinancialChart.tsx`, accountant/loans/admin-dashboard surplus pages.
- Sidebar: remove Sổ Cái (`AdminSidebar.tsx:86`) + removed-feature entries; mobile equivalents in `pages/mobile/**`.
- Replace `PaymentHistoryPage` guts with the Track A6 pages (do this **after** A6 exists).
- AccountantPage: remove the **bulk-transfer half** (D23/D27); the payroll-report half (sao kê lương tuần) is **kept and re-homed** for ketoan/giamdoc (D28) — keep `components/timesheet/{PayrollReportExportDialog,PayrollReportEmailDialog}.tsx` + `hooks/api/useExportPayrollReport` and mount them on the ketoan/giamdoc surfaces (FR-7).
- Frontend API clients: prune `services/api/ledger.service.ts` + every client hitting removed endpoints.

### Wave 2 — Transport + wiring

- Delete `bootstrap/routes_ad_banner.go`, `routes_advance_payment.go`, `routes_api_key_admin.go`, `routes_disbursement.go`, `routes_integration.go`, `routes_mcp.go`, `routes_wallet.go`, `routes_weekly_payment_fee.go`; `setupLedgerRoutes` + its group in `routes_misc.go:58+`; `setupCronRoutes` in `routes_admin.go:118-123`; drop their `routes.go` call sites (line ~70 area).
- `container.go`: remove handler/repo/service wiring for all removed stacks (Ledger `:95,492`, OnePayFeeImport, wallet, loans, ads, MCP, api keys, disbursement, flex pay, advance payment, cron handler if health pages die).
- `configs/casbin_policy.csv`: delete policies for removed endpoints (partner + admin rows referencing `/advance-payments`, `/wallet`, `/ledger`, `/api-keys`, `/mcp`, `/disbursement`, ad banners, weekly payment fees, cron-jobs). Ketoan rows already landed with PAYSUB-1344 (D28): reads mirror giamdoc; approval widened to quanly + ketoan on single entries and the pending selection per D29 (global bulk ops stay admin-only).

### Wave 3 — App services, jobs, async tasks

- Delete service dirs: `app/services/{advance_payment,flex_pay,loan,ledger,wallet_bulk,disbursement,settlement,mcpapikey,apikey,opsapikey,ad_banner,integration}` + `app/services/payroll/bulktransfer/` + `app/services/payroll/report_by_project_exporter.go` (D23 — **amended by D28: `report_by_project_exporter.go` + `report_by_project_service.go` are kept**, the weekly sao kê lương tuần exporter for FR-7); keep `app/services/payroll` core (salary computation, payrate freeze) and `app/services/excel` (billing-report export + timesheet-import parser family: monthly BCC, weekly block 2026-11-19, POURIN weekly-payment 2026-11-18, format auto-detection).
- `bootstrap/scheduler_jobs.go`: remove jobs `advance_payment_reminder`, `sync_wallet_balance`, `wallet_balance_alert` (keep its push fan-out util for A3), `loan_repayment_reminder`, `daily_receivable_reconciliation` (replaced by A5 job), `apply_payment_schedule_changes` + `apply_pending_check_in_changes` + `check_in_activation_reminder` + `prepare_check_in_roster` + `cleanup_check_in_roster` + `cancel_flexible_pending_requests` + `send_flexible_sao_ke_reminder` (check-in/pay-cycle-switch machinery, D12/D8), `vfic_funding_reminder` (front-fund), `monitor_high_error_rates` + `cleanup_old_api_metrics` (ops/api-metric surface). Keep `recompute_project_aggregates` (redefined, §6); `zalo_token_renew` removed (D20).
- Prune `initScheduler` parameter list in the same commit (it wires the removed services).
- `infra/asynq/mux.go` + `handlers.go`: unregister `TaskFlexPay*`, `TaskBulkTransfer*`, `TaskNinePay*`, `TaskWallet*` handlers and their workers; delete task constants.
- Event factories: remove `event_factory_financial.go`, `event_factory_ad_banner.go`, `event_factory_advance_payment_fee.go`, `event_factory_disbursement_fee.go`, `event_factory_weekly_payment_fee.go`; audit-subscribers of those events.
- Notification services: delete `LoanRepaymentReminderService`, `WalletBalanceAlertService` job wiring; keep generic push fan-out.
- **Redis hygiene**: after this wave, flush stale asynq tasks for removed task names (dead payloads would error on unknown-handler at enqueue time).

### Wave 4 — Domain + persistence

- Delete domain entities + repository interfaces + `infra/persistence` repos per PRD §7 drop list (`ad_banner`, `advance_payment*`, `api_key`, `mcp_api_key`, `ops_api_key`, `api_endpoint`, `api_metric`, `loan*`, `wallet/*`, `ledger.go`, `transaction*`, `settlement*`, `bulk_transfer_*`, `cron_job_status`, `disbursement_fee_schedule`, `weekly_payment_fee_schedule`, `employee_paid_activity`) + `domain/ports` entries + `bootstrap/repositories/init.go` wiring.
- Delete the `AutoMigrate` call in `bootstrap/infrastructure/init.go` entirely — `schema.sql` (§3) is the only schema authority, so boots can never resurrect dropped tables.
- Remaining timesheet writers audit: after Wave 3, the only writer of `payment_status='paid'` must be the A3 verification service — grep-verify, then remove legacy writers with the entities.
- Strip `gorm.DeletedAt` from every kept model and delete all `deleted_at IS NULL` query scopes: `db.Delete()` becomes a hard DELETE (D16). The "unique among active rows" repo invariants collapse into the plain UNIQUE constraints in `schema.sql`.

### Wave 5 — Schema swap (no destructive migration)

Replace `backend/migrations/` (001…146 + `embed.go` contents) with the single `schema.sql` from §3, applied by the deploy pipeline on first boot. Because deployment is from scratch there is nothing to ALTER or DROP: the new database simply starts at the final schema. Donor `schema_migrations` bookkeeping is not carried.
## 6. Cross-Cutting Couplings (the parts that bite)

| Coupling | Handling |
|---|---|
| `container.go` + `services/init.go` + `repositories/init.go` are wiring hubs touched by every wave | Serialize Waves 2–4 on one owner; never parallel-edit wiring files |
| `Project` aggregate recompute (`AggregateRecomputeService`, job `recompute_project_aggregates`) reads `revenue_receivable`/`revenue_paid` | Redefine aggregates in Track A (new-model fields: customer-paid totals per project); the dropped columns simply never exist in `schema.sql` (§3) |
| `PayCycleChangedEvent` + pay-cycle weekly/monthly switch | Fixed calendar (D8) kills the switch: remove event publisher/subscribers (approval, notification) with Wave 3 job removal |
| `PaymentHistoryPage` uses `components/transaction/` | Sequence: A6 replaces page content → Wave 1 deletes old components |
| Wallet balance alert = the only existing admin-wide push fan-out | Extract/reuse the fan-out for D10 **before** deleting the service in Wave 3 (Track A dependency) |
| `admin-dashboard` pages consume `ledger.service.ts` + removed KPIs | Dashboard rebuild is not in PRD scope: delete widget cards; leave a minimal dashboard (projects/employees/timesheets summaries) so the route isn't blank |
| Timesheet `failed` enum value | Document-only change (no enum migration); verification never writes it |
| Integration tests referencing removed endpoints | Rewrite per wave; new suites for upload/verify/report land with Track A |
| Assignment-window logic (`LastDate == NULL` = active; rosters, active counts, import payrate resolver) | D17: window dies — row existence + `UNIQUE(project_id, employee_id)` replaces it; `start_date` survives as the single effective-from; all `last_date` scopes deleted in Wave 4 |
| Deleting an employee with salary history | D18: FKs to `timesheets`/`employee_pay_cycles` are RESTRICT (default in `schema.sql`) — the API must surface a clear "nhân viên có dữ liệu, không thể xóa" error; no cascade from user actions |
| Soft delete is woven through kept models (uniques like `unique_cccd_deleted_at`, generated `live_from_date`/`active_key`, query scopes) | Hard-delete conversion (D16) lands in Wave 4 with the model strip; `schema.sql` ships the rebuilt plain UNIQUEs; audit_logs remains the trail for deletions |

## 7. Ticket → Wave Mapping

| Ticket | Wave/Track |
|---|---|
| PAYSUB-1 Schema rebuild | §3 consolidated `schema.sql` (24 tables) + Wave 4 AutoMigrate removal |
| PAYSUB-2 (FR-1) | Track A A4 |
| PAYSUB-3 (FR-2) | Track A A2–A3 |
| PAYSUB-4 (FR-3) | Track A A5 |
| PAYSUB-5 (FR-4) | Track A A1 |
| PAYSUB-6 (FR-5) | **CANCELED (D21)** — no customer/partner concept, `partner_company` removed |
| PAYSUB-7 payment rails | Waves 1–4 slices; 2% fee logic (D14) dies in Wave 3/4 |
| PAYSUB-8 credit & check-in | Waves 1–4 slices; wallet push fan-out extracted in Track A first |
| PAYSUB-9 misc surface | Waves 1–2 slices; dropped settings keys simply never exist in `schema.sql` (only the fee key is seeded) |

## 8. Safety & Rollback

- Repo works on main directly (repo rule): each wave = focused conventional commits on main, tag `paysub-wave-N` after the gate passes — revert unit per wave, not per file.
- No destructive DB steps exist: the schema is a fresh `CREATE` script; rollback = `git revert` of the wave (drop + recreate the database from `schema.sql` at any time).
- Redis: flush asynq queues for removed task names at Wave 3 (stale payloads have no handler).
- Board protocol: each PAYSUB ticket moves only with an evidence comment (commit hashes, gate outputs, screenshots for UI) per the completion rules in `AGENTS.md`.

## 9. Explicit Non-Goals

- No payment/invoice/VAT integration (D9). No multi-tenancy (D7). No calendar configurability (D8). No data migration from any production system (greenfield). No ledger replacement — the billing report is the only financial artifact (D13/D14).
