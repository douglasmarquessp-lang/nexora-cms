-- 000003_add_audit_log.down.sql
-- Rollback audit_log additions.
--
-- NOTE: audit_log was first created in 000001 as a partitioned table.
-- 000003's CREATE TABLE IF NOT EXISTS is a no-op when 000001 ran first.
-- We only drop the indexes that this migration created; the table itself
-- (whether partitioned or non-partitioned) is owned by 000001.down.
--
-- WARNING: If 000001 was never applied, the audit_log table was created
-- by this migration as a non-partitioned table. In that scenario, this
-- down migration will leave an orphaned audit_log table. Manual
-- DROP TABLE IF EXISTS audit_log is required in that case.

DROP INDEX IF EXISTS idx_audit_log_entity;
DROP INDEX IF EXISTS idx_audit_log_created;
DROP INDEX IF EXISTS idx_audit_log_action;
DROP INDEX IF EXISTS idx_audit_log_user;
