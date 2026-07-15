-- 000003_add_audit_log.down.sql

DROP INDEX IF EXISTS idx_audit_log_entity;
DROP INDEX IF EXISTS idx_audit_log_created;
DROP INDEX IF EXISTS idx_audit_log_action;
DROP INDEX IF EXISTS idx_audit_log_user;
DROP TABLE IF EXISTS audit_log;
