-- PAYSUB consolidated schema — backend/migrations/schema.sql
-- Verbatim from TechnicalDesign.md §3 "Final Schema (consolidated)" sql block
-- (repo root), extracted 2026-11-20. This script is the ONLY schema authority:
-- numbered donor migrations 001-146 and the gorm embed runner are removed;
-- apply once on first deploy. gorm AutoMigrate must stay OFF.
-- Deviation from the §3 block as committed: CREATE TABLE `timesheet_edit_requests`
-- is NOT carried here — the §2 removal matrix (D25) drops that table and states
-- schema.sql already excludes it ("keep it that way"); no Go code references it.
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
  `role` enum('admin','quanly','ketoan','giamdoc','nhanvien') NOT NULL DEFAULT 'nhanvien',
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
  KEY `idx_users_last_login` (`last_login`),
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
  CONSTRAINT `fk_employees_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
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
  CONSTRAINT `fk_project_employees_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`),
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
  CONSTRAINT `fk_project_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`),
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
  CONSTRAINT `fk_employee_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
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
  CONSTRAINT `fk_payrates_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`),
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
  KEY `idx_timesheets_payment_date` (`payment_status`,`date`),
  KEY `idx_timesheets_employee_payment_paid` (`employee_id`,`payment_status`,`paid_at`),
  CONSTRAINT `fk_timesheets_approved_user` FOREIGN KEY (`approved_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_timesheets_created_user` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_timesheets_employee` FOREIGN KEY (`employee_id`) REFERENCES `employees` (`id`),
  CONSTRAINT `fk_timesheets_payrate` FOREIGN KEY (`payrate_id`) REFERENCES `payrates` (`id`),
  CONSTRAINT `fk_timesheets_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`),KEY `idx_timesheets_employee_date` (`employee_id`,`date`),
KEY `idx_timesheets_date_status` (`date`,`timesheet_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

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
  `parsed_rows` int NOT NULL DEFAULT '0' COMMENT 'rows parsed from the file (dashboard recon block)',
  `matched_rows` int NOT NULL DEFAULT '0' COMMENT 'rows matched to approved timesheets (dashboard recon block)',
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
  `asset_id` bigint unsigned DEFAULT NULL,
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
